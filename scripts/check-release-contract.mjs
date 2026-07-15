import { readFileSync } from "node:fs";

const requestedTag = process.argv[2] ?? "v1.0.0";
const version = requestedTag.replace(/^v/, "");
const files = ["package.json", "web/package.json", "desktop/package.json", "desktop/src-tauri/tauri.conf.json"];
const failures = [];
for (const file of files) {
  const value = JSON.parse(readFileSync(file, "utf8"));
  if (value.version !== version) failures.push(`${file}: version ${value.version} does not match ${version}`);
}
const cargo = readFileSync("desktop/src-tauri/Cargo.toml", "utf8");
if (!cargo.includes(`version = "${version}"`)) failures.push(`desktop/src-tauri/Cargo.toml does not match ${version}`);
const openapi = JSON.parse(readFileSync("docs/reference/openapi.yaml", "utf8"));
if (openapi.info?.version !== version) failures.push(`OpenAPI version does not match ${version}`);
if (!readFileSync("CHANGELOG.md", "utf8").includes("## Unreleased")) failures.push("CHANGELOG.md must retain an Unreleased section");
const workflow = readFileSync(".github/workflows/release.yml", "utf8");
for (const required of ["sbom.spdx.json", "sbom.cyclonedx.json", "check-license-policy.mjs", "SHA256SUMS", "cosign sign", "attest-build-provenance"]) {
  if (!workflow.includes(required)) failures.push(`release workflow is missing ${required}`);
}
if (/tauri-action[\s\S]*tagName:/.test(workflow)) failures.push("desktop build must not create a GitHub release before the protected publish job");
if (!workflow.includes("environment: production-release")) failures.push("release publish job must use the production-release environment");
if (failures.length > 0) {
  console.error(`check-release-contract: ${failures.length} violation(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}
console.log(`check-release-contract: ok version=${version}`);
