#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const packageDirs = [
  "npm/linux-x64",
  "npm/linux-arm64",
  "npm/darwin-x64",
  "npm/darwin-arm64",
  "."
];

for (const packageDir of packageDirs) {
  console.log(`publishing ${packageDir}`);
  const result = spawnSync("npm", ["publish", "--provenance", "--access", "public"], {
    cwd: path.join(root, packageDir),
    stdio: "inherit"
  });
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}
