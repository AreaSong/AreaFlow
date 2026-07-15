import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { execFileSync } from "node:child_process";

const lockfiles = ["package-lock.json", "web/package-lock.json", "desktop/package-lock.json"];
const denied = /(?:^|\s|\()(?:AGPL|GPL|SSPL|BUSL|Commons-Clause|UNLICENSED)(?:-|\s|\)|$)/i;
const failures = [];
let goModules = 0;

for (const file of lockfiles) {
  const lock = JSON.parse(readFileSync(file, "utf8"));
  for (const [name, entry] of Object.entries(lock.packages ?? {})) {
    if (!name || !entry || entry.link) continue;
    const license = String(entry.license ?? "").trim();
    if (!license) failures.push(`${file}:${name}: missing license metadata`);
    else if (denied.test(license)) failures.push(`${file}:${name}: denied license ${license}`);
  }
}

const moduleLines = execFileSync("go", ["list", "-m", "-f", "{{if not .Main}}{{.Path}}\t{{.Dir}}{{end}}", "all"], { encoding: "utf8" });
for (const line of moduleLines.split("\n")) {
  if (!line.trim()) continue;
  const [modulePath, moduleDir] = line.split("\t");
  if (!modulePath || !moduleDir) continue;
  goModules++;
  const licenseFile = findLicenseFile(moduleDir);
  if (!licenseFile) {
    failures.push(`go:${modulePath}: missing license file`);
    continue;
  }
  const text = readFileSync(licenseFile, "utf8").slice(0, 12000);
  const header = text.slice(0, 1000);
  if (/GNU AFFERO GENERAL PUBLIC LICENSE|Server Side Public License|Business Source License|Commons Clause/i.test(text)) {
    failures.push(`go:${modulePath}: denied license in ${licenseFile}`);
  } else if (/GNU GENERAL PUBLIC LICENSE/i.test(header) && !/GNU LESSER GENERAL PUBLIC LICENSE/i.test(header)) {
    failures.push(`go:${modulePath}: denied GPL license in ${licenseFile}`);
  }
}

if (failures.length > 0) {
  console.error(`check-license-policy: ${failures.length} violation(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}
console.log(`check-license-policy: ok lockfiles=${lockfiles.length} go_modules=${goModules}`);

function findLicenseFile(moduleDir) {
  let current = moduleDir;
  for (let depth = 0; depth < 4; depth++) {
    if (!existsSync(current) || !statSync(current).isDirectory()) return "";
    const candidate = readdirSync(current).find((name) => /^(LICENSE|LICENCE|COPYING)(\.|$)/i.test(name));
    if (candidate) return join(current, candidate);
    const parent = dirname(current);
    if (parent === current || !parent.includes("/pkg/mod/")) break;
    current = parent;
  }
  return "";
}
