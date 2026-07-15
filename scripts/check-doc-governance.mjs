import { readdirSync } from "node:fs";
import { extname, join, relative, sep } from "node:path";

const root = process.cwd();
const docsRoot = join(root, "docs");
const allowedTopLevelFiles = new Set(["README.md", "roadmap.md"]);
const allowedDirectories = new Set([
  "adr",
  "architecture",
  "concepts",
  "development",
  "getting-started",
  "guides",
  "history",
  "operations",
  "reference",
]);
const forbiddenStageName = /(?:^|[-_])(?:phase-?\d+|milestone|progress|evidence|implementation-plan|completion-report|completion-summary|verification-log|test-log)(?:[-_.]|$)/i;
const failures = [];

for (const file of collectMarkdownFiles(docsRoot)) {
  const path = relative(docsRoot, file);
  const parts = path.split(sep);

  if (parts.length === 1 && !allowedTopLevelFiles.has(path)) {
    failures.push(`${path}: docs 顶层只允许 README.md 与 roadmap.md`);
    continue;
  }

  if (parts.length > 1 && !allowedDirectories.has(parts[0])) {
    failures.push(`${path}: 未知文档分类 ${parts[0]}`);
    continue;
  }

  if (parts[0] === "history" && parts.length < 3 && parts[1] !== "README.md") {
    failures.push(`${path}: 历史材料必须位于 docs/history/<release>/**`);
    continue;
  }

  if (parts[0] !== "history" && forbiddenStageName.test(parts.at(-1))) {
    failures.push(`${path}: 阶段性文档必须归档到 docs/history/<release>/**`);
  }
}

if (failures.length > 0) {
  console.error(`check-doc-governance: ${failures.length} violation(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log("check-doc-governance: ok");

function collectMarkdownFiles(directory) {
  const files = [];
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) files.push(...collectMarkdownFiles(path));
    else if (entry.isFile() && extname(entry.name).toLowerCase() === ".md") files.push(path);
  }
  return files;
}
