#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { mkdir } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const distDir = path.join(root, "dist", "npm");
const version = versionFromTag() ?? versionFromPackage();

const targets = [
  { name: "linux-x64", goos: "linux", goarch: "amd64" },
  { name: "linux-arm64", goos: "linux", goarch: "arm64" },
  { name: "darwin-x64", goos: "darwin", goarch: "amd64" },
  { name: "darwin-arm64", goos: "darwin", goarch: "arm64" }
];

await mkdir(distDir, { recursive: true });

for (const target of targets) {
  const outDir = path.join(distDir, target.name);
  await mkdir(outDir, { recursive: true });
  const outPath = path.join(outDir, "octx");
  console.log(`building ${target.name}`);
  const ldflags = `-s -w -X github.com/ninj4dkill4/octx/internal/version.Version=${version}`;
  const result = spawnSync("go", ["build", "-trimpath", "-ldflags", ldflags, "-o", outPath, "./cmd/octx"], {
    cwd: root,
    env: {
      ...process.env,
      CGO_ENABLED: "0",
      GOOS: target.goos,
      GOARCH: target.goarch
    },
    stdio: "inherit"
  });
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

function versionFromTag() {
  const ref = process.env.GITHUB_REF_NAME;
  if (!ref) {
    return null;
  }
  const match = /^v(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$/.exec(ref);
  if (!match) {
    return null;
  }
  return match[1];
}

function versionFromPackage() {
  const packagePath = path.join(root, "package.json");
  const packageJSON = JSON.parse(readFileSync(packagePath, "utf8"));
  return packageJSON.version;
}
