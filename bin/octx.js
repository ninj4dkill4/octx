#!/usr/bin/env node
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(import.meta.url);

const platformPackages = {
  "darwin-arm64": "@ninj4dkill4/octx-darwin-arm64",
  "darwin-x64": "@ninj4dkill4/octx-darwin-x64",
  "linux-arm64": "@ninj4dkill4/octx-linux-arm64",
  "linux-x64": "@ninj4dkill4/octx-linux-x64"
};

const target = `${process.platform}-${process.arch}`;
const packageName = platformPackages[target];

if (!packageName) {
  console.error(`Unsupported platform for @ninj4dkill4/octx: ${target}`);
  process.exit(1);
}

let binaryPath;
try {
  const packageJson = require.resolve(`${packageName}/package.json`);
  binaryPath = path.join(path.dirname(packageJson), "bin", process.platform === "win32" ? "octx.exe" : "octx");
} catch {
  console.error(`Missing optional dependency ${packageName}. Reinstall @ninj4dkill4/octx.`);
  process.exit(1);
}

if (!existsSync(binaryPath)) {
  console.error(`Missing octx binary at ${binaryPath}. Reinstall @ninj4dkill4/octx.`);
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  env: process.env
});

child.on("error", (error) => {
  console.error(error.message);
  process.exit(1);
});

for (const signal of ["SIGINT", "SIGTERM", "SIGHUP"]) {
  process.on(signal, () => {
    if (!child.killed) {
      child.kill(signal);
    }
  });
}

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
