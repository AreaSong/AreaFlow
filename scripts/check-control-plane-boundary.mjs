import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";

const failures = [];
const appFiles = trackedFiles("internal/app", "*.go");
for (const file of appFiles) {
  const source = readFileSync(file, "utf8");
  if (/\.(?:Exec|Query|QueryRow|Begin)\s*\(/.test(source)) {
    failures.push(`${file}: CLI must call domain command/query APIs instead of pgx SQL methods`);
  }
}

for (const file of [...trackedFiles("web/src", "*.ts"), ...trackedFiles("web/src", "*.tsx"), ...trackedFiles("desktop/src", "*.ts")]) {
  const source = readFileSync(file, "utf8");
  if (/from\s+["'](?:pg|postgres|postgres-js)["']|require\(["'](?:pg|postgres|postgres-js)["']\)|AREAFLOW_DATABASE_URL/.test(source)) {
    failures.push(`${file}: Web/Desktop must use the AreaFlow API instead of database state`);
  }
}

const appSource = appFiles.map((file) => readFileSync(file, "utf8")).join("\n");
for (const required of ["store.RunWorkerOnce(", "store.ApplyStatusProjection(", "store.CreateWorkflowVersion(", "store.CreateApprovalRecord("]) {
  if (!appSource.includes(required)) failures.push(`CLI command boundary is missing ${required}`);
}
const webAPI = readFileSync("web/src/api.ts", "utf8");
if (!webAPI.includes('const API_BASE = "/api/v1"')) failures.push("Web API base must remain /api/v1");

if (failures.length > 0) {
  console.error(`check-control-plane-boundary: ${failures.length} violation(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}
console.log(`check-control-plane-boundary: ok app_files=${appFiles.length}`);

function trackedFiles(directory, glob) {
  const output = execFileSync("rg", ["--files", "-g", glob, directory], { encoding: "utf8" }).trim();
  return output ? output.split("\n") : [];
}
