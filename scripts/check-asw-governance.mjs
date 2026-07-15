import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

const root = process.cwd();
const adoptionPath = resolve(root, "governance/asw-ewf-001-adoption.yaml");
const record = JSON.parse(readFileSync(adoptionPath, "utf8"));
const failures = [];

if (record.schema_version !== 1) failures.push("schema_version must be 1");
if (record.spec?.id !== "ASW-EWF-001" || record.spec?.version !== "1.0.0") {
  failures.push("spec must be ASW-EWF-001@1.0.0");
}

const sourcePath = resolve(root, record.spec?.source ?? "");
if (!existsSync(sourcePath)) {
  failures.push(`missing spec source: ${record.spec?.source ?? ""}`);
} else {
  const digest = createHash("sha256").update(readFileSync(sourcePath)).digest("hex");
  if (digest !== record.spec.sha256) failures.push(`spec sha256 mismatch: ${digest}`);
}

const roleKeys = new Set(Object.keys(record.roles ?? {}));
const ids = new Set();
for (const control of record.controls ?? []) {
  validateID(control.id, "control");
  if (!roleKeys.has(control.owner)) failures.push(`${control.id}: unknown owner role ${control.owner}`);
  if (!Array.isArray(control.evidence)) failures.push(`${control.id}: evidence must be an array`);
  if (control.status === "satisfied" && control.evidence.length === 0) failures.push(`${control.id}: satisfied control requires evidence`);
  if (control.status === "partial" && !control.close_when) failures.push(`${control.id}: partial control requires close_when`);
  for (const evidence of control.evidence ?? []) {
    if (!existsSync(resolve(root, evidence))) failures.push(`${control.id}: missing evidence ${evidence}`);
  }
}

for (const dependency of record.external_dependencies ?? []) {
  validateID(dependency.id, "external dependency");
  if (!roleKeys.has(dependency.owner)) failures.push(`${dependency.id}: unknown owner role ${dependency.owner}`);
  if (dependency.status !== "blocked_external") failures.push(`${dependency.id}: external dependency cannot be marked complete by repository evidence`);
  if (!dependency.condition || !dependency.verification) failures.push(`${dependency.id}: condition and verification are required`);
}

if ((record.controls ?? []).length < 9) failures.push("G0-G8 controls are required");
for (let gate = 0; gate <= 8; gate += 1) {
  if (!ids.has(`G${gate}`)) failures.push(`missing G${gate} control`);
}

if (failures.length > 0) {
  console.error(`check-asw-governance: ${failures.length} violation(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log(`check-asw-governance: ok controls=${record.controls.length} external=${record.external_dependencies.length}`);

function validateID(id, kind) {
  if (!id || ids.has(id)) failures.push(`${kind} id is missing or duplicated: ${id ?? ""}`);
  else ids.add(id);
}
