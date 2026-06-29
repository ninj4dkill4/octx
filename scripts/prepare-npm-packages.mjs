#!/usr/bin/env node
import { chmod, copyFile, mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const rootPackagePath = path.join(root, "package.json");
const rootPackage = JSON.parse(await readFile(rootPackagePath, "utf8"));
const version = versionFromTag() ?? rootPackage.version;

rootPackage.version = version;
for (const dependency of Object.keys(rootPackage.optionalDependencies ?? {})) {
  rootPackage.optionalDependencies[dependency] = version;
}
await writeJSON(rootPackagePath, rootPackage);

const platformPackages = [
  { dir: "linux-x64", name: "@ninj4dkill4/octx-linux-x64" },
  { dir: "linux-arm64", name: "@ninj4dkill4/octx-linux-arm64" },
  { dir: "darwin-x64", name: "@ninj4dkill4/octx-darwin-x64" },
  { dir: "darwin-arm64", name: "@ninj4dkill4/octx-darwin-arm64" }
];

for (const platformPackage of platformPackages) {
  const packageDir = path.join(root, "npm", platformPackage.dir);
  const packagePath = path.join(packageDir, "package.json");
  const manifest = JSON.parse(await readFile(packagePath, "utf8"));
  manifest.version = version;
  await writeJSON(packagePath, manifest);

  const binDir = path.join(packageDir, "bin");
  await mkdir(binDir, { recursive: true });
  const source = path.join(root, "dist", "npm", platformPackage.dir, "octx");
  const destination = path.join(binDir, "octx");
  await copyFile(source, destination);
  await chmod(destination, 0o755);
}

function versionFromTag() {
  const ref = process.env.GITHUB_REF_NAME;
  if (!ref) {
    return null;
  }
  const match = /^v(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$/.exec(ref);
  if (!match) {
    throw new Error(`GITHUB_REF_NAME must look like vX.Y.Z, got ${ref}`);
  }
  return match[1];
}

async function writeJSON(file, value) {
  await writeFile(file, `${JSON.stringify(value, null, 2)}\n`);
}
