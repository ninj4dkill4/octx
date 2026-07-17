package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ninj4dkill4/octx/internal/config"
)

type Level string

const (
	OK    Level = "OK"
	Warn  Level = "WARN"
	Error Level = "ERROR"
)

type Result struct {
	Level   Level
	Project string
	Check   string
	Message string
}

type Report struct {
	Results []Result
}

func (r Report) ErrorCount() int {
	count := 0
	for _, result := range r.Results {
		if result.Level == Error {
			count++
		}
	}
	return count
}

func (r Report) HasErrors() bool {
	return r.ErrorCount() > 0
}

type Options struct {
	Paths      config.Paths
	Env        map[string]string
	LookPath   func(string) (string, error)
	RunCommand func(string, ...string) (string, error)
	Executable func() (string, error)
}

func Run(opts Options) Report {
	opts = withDefaults(opts)
	checker := checker{opts: opts}
	checker.run()
	return Report{Results: checker.results}
}

type checker struct {
	opts    Options
	cfg     config.Config
	cfgOK   bool
	results []Result
}

func (c *checker) run() {
	c.checkConfig()
	if !c.cfgOK {
		c.checkExecutable()
		return
	}
	c.checkSSH()
	c.checkKubeconfig()
	c.checkEnv()
	c.checkExternalProfiles()
	c.checkExecutable()
}

func (c *checker) add(level Level, check, message string) {
	c.results = append(c.results, Result{
		Level:   level,
		Check:   check,
		Message: message,
	})
}

func (c *checker) addProject(level Level, project, check, message string) {
	c.results = append(c.results, Result{
		Level:   level,
		Project: project,
		Check:   check,
		Message: message,
	})
}

func (c *checker) checkConfig() {
	cfg, err := config.LoadConfig(c.opts.Paths.ConfigFile)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			c.add(Error, "config", fmt.Sprintf("config not found at %s; run `octx init` first", c.opts.Paths.ConfigFile))
			return
		}
		c.add(Error, "config", fmt.Sprintf("config invalid at %s: %v", c.opts.Paths.ConfigFile, err))
		return
	}

	c.cfg = cfg
	c.cfgOK = true
	c.add(OK, "config", fmt.Sprintf("loaded %s", c.opts.Paths.ConfigFile))
	if len(cfg.Projects) == 0 {
		c.add(Warn, "config", "no projects configured")
	}
}

func (c *checker) checkSSH() {
	hasSSHConfig := false
	for _, project := range c.cfg.Projects {
		if project.SSHConfig == "" {
			continue
		}
		hasSSHConfig = true
		path := config.ExpandPath(project.SSHConfig)
		if _, err := os.Stat(path); err != nil {
			c.addProject(Warn, project.Code, "ssh", fmt.Sprintf("ssh_config %s: %v", path, err))
			continue
		}
		c.addProject(OK, project.Code, "ssh", "ssh_config exists")
	}
	if hasSSHConfig {
		c.checkLegacySSHInclude()
	}

	c.checkShellSSHConfig()
}

func (c *checker) checkLegacySSHInclude() {
	path := filepath.Join(c.opts.Env["HOME"], ".ssh", "config")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	if containsSSHInclude(string(data), c.opts.Paths.SSHCurrent, c.opts.Env["HOME"]) {
		c.add(Warn, "ssh", fmt.Sprintf("%s still includes legacy %s; remove it when using OCTX_SSH_CONFIG", path, c.opts.Paths.SSHCurrent))
	}
}

func (c *checker) checkShellSSHConfig() {
	path := c.opts.Env["OCTX_SSH_CONFIG"]
	if path == "" {
		return
	}
	path = config.ExpandPath(path)
	if _, err := os.Stat(path); err != nil {
		c.add(Warn, "ssh", fmt.Sprintf("OCTX_SSH_CONFIG %s: %v", path, err))
		return
	}
	c.add(OK, "ssh", fmt.Sprintf("OCTX_SSH_CONFIG points to %s", path))
}

func (c *checker) checkKubeconfig() {
	for _, project := range c.cfg.Projects {
		if project.Kubeconfig == "" {
			continue
		}
		path := config.ExpandPath(project.Kubeconfig)
		if _, err := os.Stat(path); err != nil {
			c.addProject(Warn, project.Code, "kube", fmt.Sprintf("kubeconfig %s: %v", path, err))
			continue
		}
		c.addProject(OK, project.Code, "kube", "kubeconfig exists")
	}
}

func (c *checker) checkEnv() {
	shellProject := c.opts.Env["OPSCTX_PROJECT"]
	if shellProject == "" || shellProject == config.UnsetProjectCode {
		c.add(OK, "shell", "context unset")
		return
	}
	project, ok := c.cfg.FindProject(shellProject)
	if !ok {
		c.add(Warn, "shell", fmt.Sprintf("OPSCTX_PROJECT=%q is not in config", shellProject))
		return
	}

	c.add(OK, "shell", fmt.Sprintf("context is %s", project.Code))
	c.checkEnvValue("OPSCTX_PROJECT", project.Code)
	c.checkEnvValue("AWS_PROFILE", project.AWSProfile)
	c.checkEnvValue("ALIBABA_CLOUD_PROFILE", project.AliyunProfile)
	c.checkEnvValue("CODEX_PROFILE", project.CodexProfile)
	c.checkEnvValue("CLOUDSDK_ACTIVE_CONFIG_NAME", project.GCloudConfig)
	c.checkPathEnvValue("AZURE_CONFIG_DIR", project.AzureConfigDir)
	c.checkPathEnvValue("KUBECONFIG", project.Kubeconfig)
}

func (c *checker) checkEnvValue(key, want string) {
	got, ok := c.opts.Env[key]
	if want == "" {
		if ok && got != "" {
			c.add(Warn, "env", fmt.Sprintf("%s=%q, want unset", key, got))
		}
		return
	}
	if !ok || got != want {
		c.add(Warn, "env", fmt.Sprintf("%s=%q, want %q", key, got, want))
		return
	}
	c.add(OK, "env", fmt.Sprintf("%s matches", key))
}

func (c *checker) checkPathEnvValue(key, want string) {
	if want != "" {
		want = config.ExpandPath(want)
	}
	c.checkEnvValue(key, want)
}

func (c *checker) checkExternalProfiles() {
	c.checkAWSProfiles()
	c.checkAliyunProfiles()
	c.checkCodexProfiles()
	c.checkGCloudConfigs()
	c.checkAzureConfigDirs()
}

func (c *checker) checkAWSProfiles() {
	projects := projectsWithProfile(c.cfg.Projects, func(project config.Project) string {
		return project.AWSProfile
	})
	if len(projects) == 0 {
		return
	}
	cliPath, err := c.opts.LookPath("aws")
	if err != nil {
		for _, project := range projects {
			c.addProject(Warn, project.Code, "aws", "aws CLI not found; skipping AWS profile validation")
		}
		return
	}
	c.add(OK, "aws", fmt.Sprintf("CLI found at %s", cliPath))
	output, err := c.opts.RunCommand("aws", "configure", "list-profiles")
	if err != nil {
		for _, project := range projects {
			c.addProject(Warn, project.Code, "aws", fmt.Sprintf("could not list AWS profiles: %v", err))
		}
		return
	}
	available := parseLineProfiles(output)
	for _, project := range projects {
		profile := project.AWSProfile
		if !available[profile] {
			c.addProject(Warn, project.Code, "aws", fmt.Sprintf("profile %q not found", profile))
			continue
		}
		c.addProject(OK, project.Code, "aws", fmt.Sprintf("profile %q exists", profile))
	}
}

func (c *checker) checkAliyunProfiles() {
	projects := projectsWithProfile(c.cfg.Projects, func(project config.Project) string {
		return project.AliyunProfile
	})
	if len(projects) == 0 {
		return
	}
	cliPath, err := c.opts.LookPath("aliyun")
	if err != nil {
		for _, project := range projects {
			c.addProject(Warn, project.Code, "aliyun", "aliyun CLI not found; skipping Aliyun profile validation")
		}
		return
	}
	c.add(OK, "aliyun", fmt.Sprintf("CLI found at %s", cliPath))
	output, err := c.opts.RunCommand("aliyun", "configure", "list")
	if err != nil {
		for _, project := range projects {
			c.addProject(Warn, project.Code, "aliyun", fmt.Sprintf("could not list Aliyun profiles: %v", err))
		}
		return
	}
	available := parseAliyunProfiles(output)
	for _, project := range projects {
		profile := project.AliyunProfile
		if !available[profile] {
			c.addProject(Warn, project.Code, "aliyun", fmt.Sprintf("profile %q not found", profile))
			continue
		}
		c.addProject(OK, project.Code, "aliyun", fmt.Sprintf("profile %q exists", profile))
	}
}

func (c *checker) checkCodexProfiles() {
	projects := projectsWithProfile(c.cfg.Projects, func(project config.Project) string {
		return project.CodexProfile
	})
	if len(projects) == 0 {
		return
	}
	base := c.opts.Env["CODEX_HOME"]
	if base == "" {
		base = filepath.Join(c.opts.Env["HOME"], ".codex")
	}
	for _, project := range projects {
		profile := project.CodexProfile
		path := filepath.Join(base, profile+".config.toml")
		if _, err := os.Stat(path); err != nil {
			c.addProject(Warn, project.Code, "codex", fmt.Sprintf("profile %q file not found at %s", profile, path))
			continue
		}
		c.addProject(OK, project.Code, "codex", fmt.Sprintf("profile %q exists", profile))
	}
}

func (c *checker) checkGCloudConfigs() {
	projects := projectsWithProfile(c.cfg.Projects, func(project config.Project) string {
		return project.GCloudConfig
	})
	if len(projects) == 0 {
		return
	}
	cliPath, err := c.opts.LookPath("gcloud")
	if err != nil {
		for _, project := range projects {
			c.addProject(Warn, project.Code, "gcloud", "gcloud CLI not found; skipping GCloud config validation")
		}
		return
	}
	c.add(OK, "gcloud", fmt.Sprintf("CLI found at %s", cliPath))
	output, err := c.opts.RunCommand("gcloud", "config", "configurations", "list", "--format=value(name)")
	if err != nil {
		for _, project := range projects {
			c.addProject(Warn, project.Code, "gcloud", fmt.Sprintf("could not list GCloud configurations: %v", err))
		}
		return
	}
	available := parseLineProfiles(output)
	for _, project := range projects {
		name := project.GCloudConfig
		if !available[name] {
			c.addProject(Warn, project.Code, "gcloud", fmt.Sprintf("configuration %q not found", name))
			continue
		}
		c.addProject(OK, project.Code, "gcloud", fmt.Sprintf("configuration %q exists", name))
	}
}

func (c *checker) checkAzureConfigDirs() {
	projects := projectsWithProfile(c.cfg.Projects, func(project config.Project) string {
		return project.AzureConfigDir
	})
	if len(projects) == 0 {
		return
	}
	cliPath, err := c.opts.LookPath("az")
	if err != nil {
		for _, project := range projects {
			c.addProject(Warn, project.Code, "azure", "az CLI not found; skipping Azure config validation")
		}
		return
	}
	c.add(OK, "azure", fmt.Sprintf("CLI found at %s", cliPath))
	for _, project := range projects {
		dir := config.ExpandPath(project.AzureConfigDir)
		info, err := os.Stat(dir)
		if err != nil {
			c.addProject(Warn, project.Code, "azure", fmt.Sprintf("config dir %s: %v", dir, err))
			continue
		}
		if !info.IsDir() {
			c.addProject(Warn, project.Code, "azure", fmt.Sprintf("config dir %s is not a directory", dir))
			continue
		}
		configFile := filepath.Join(dir, "config")
		if _, err := os.Stat(configFile); err != nil {
			c.addProject(Warn, project.Code, "azure", fmt.Sprintf("config file %s: %v", configFile, err))
			continue
		}
		c.addProject(OK, project.Code, "azure", fmt.Sprintf("config dir %s exists", dir))
	}
}

func (c *checker) checkExecutable() {
	executable, err := c.opts.Executable()
	if err != nil {
		c.add(Warn, "binary", fmt.Sprintf("could not determine current executable: %v", err))
		return
	}
	c.add(OK, "binary", fmt.Sprintf("running %s", executable))

	resolved, err := findInPath("octx", c.opts.Env["PATH"])
	if err != nil {
		c.add(Warn, "binary", "octx not found in PATH")
		return
	}
	if !samePath(resolved, executable) {
		if isNPMWrapperForBinary(resolved, executable) {
			c.add(OK, "binary", "PATH resolves to npm launcher for the running octx binary")
			return
		}
		c.add(Warn, "binary", fmt.Sprintf("PATH resolves octx to %s, running %s", resolved, executable))
		return
	}
	c.add(OK, "binary", "PATH resolves to the running octx binary")
}

func withDefaults(opts Options) Options {
	if opts.Env == nil {
		opts.Env = envMap()
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}
	if opts.RunCommand == nil {
		opts.RunCommand = runCommand
	}
	if opts.Executable == nil {
		opts.Executable = os.Executable
	}
	return opts
}

func envMap() map[string]string {
	env := make(map[string]string)
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("%s timed out", name)
	}
	if err != nil {
		return string(output), fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func uniqueProfiles(projects []config.Project, profile func(config.Project) string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, project := range projects {
		value := profile(project)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func projectsWithProfile(projects []config.Project, profile func(config.Project) string) []config.Project {
	var result []config.Project
	for _, project := range projects {
		if profile(project) != "" {
			result = append(result, project)
		}
	}
	return result
}

func parseLineProfiles(output string) map[string]bool {
	profiles := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			profiles[line] = true
		}
	}
	return profiles
}

func parseAliyunProfiles(output string) map[string]bool {
	profiles := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.Contains(line, "Credential") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSuffix(fields[0], " *")
		name = strings.TrimSuffix(name, "*")
		if name != "" {
			profiles[name] = true
		}
	}
	return profiles
}

func containsSSHInclude(content, currentPath, home string) bool {
	want := cleanSSHPath(currentPath, home)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(fields[0], "Include") {
			continue
		}
		for _, field := range fields[1:] {
			if strings.HasPrefix(field, "#") {
				break
			}
			if cleanSSHPath(field, home) == want {
				return true
			}
		}
	}
	return false
}

func cleanSSHPath(path, home string) string {
	path = strings.Trim(path, `"'`)
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return config.ExpandPath(path)
}

func findInPath(name, pathValue string) (string, error) {
	for _, dir := range filepath.SplitList(pathValue) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", os.ErrNotExist
}

func samePath(a, b string) bool {
	aEval, aErr := filepath.EvalSymlinks(a)
	bEval, bErr := filepath.EvalSymlinks(b)
	if aErr == nil {
		a = aEval
	}
	if bErr == nil {
		b = bEval
	}
	aAbs, aErr := filepath.Abs(a)
	bAbs, bErr := filepath.Abs(b)
	if aErr == nil {
		a = aAbs
	}
	if bErr == nil {
		b = bAbs
	}
	return a == b
}

func isNPMWrapperForBinary(wrapperPath, binaryPath string) bool {
	wrapperEval, err := filepath.EvalSymlinks(wrapperPath)
	if err == nil {
		wrapperPath = wrapperEval
	}
	wrapperPath = filepath.ToSlash(wrapperPath)
	binaryPath = filepath.ToSlash(binaryPath)
	return strings.HasSuffix(wrapperPath, "/@ninj4dkill4/octx/bin/octx.js") &&
		strings.Contains(binaryPath, "/@ninj4dkill4/octx/node_modules/@ninj4dkill4/octx-") &&
		strings.HasSuffix(binaryPath, "/bin/octx")
}
