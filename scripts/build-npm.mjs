#!/usr/bin/env node
import { mkdir } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const distDir = path.join(root, "dist", "npm");

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
  const result = spawnSync("go", ["build", "-trimpath", "-ldflags", "-s -w", "-o", outPath, "./cmd/octx"], {
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
