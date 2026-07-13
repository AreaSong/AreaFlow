import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, extname, join, normalize, resolve } from "node:path";

const root = process.cwd();
const ignoredDirectories = new Set([".git", ".playwright-cli", "node_modules", "dist"]);
const markdownFiles = collectMarkdownFiles(root);
const failures = [];

for (const file of markdownFiles) {
  const content = readFileSync(file, "utf8");
  for (const target of markdownTargets(content)) {
    const path = decodeTarget(target);
    if (!path || isExternal(path) || path.startsWith("#")) continue;

    const filePart = path.split("#", 1)[0].split("?", 1)[0];
    if (!filePart) continue;

    const candidate = resolve(dirname(file), filePart);
    if (!existsAsDocumentTarget(candidate)) {
      failures.push(`${relative(file)} -> ${target}`);
    }
  }
}

if (failures.length > 0) {
  console.error(`check-doc-links: ${failures.length} broken relative link(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log(`check-doc-links: ok files=${markdownFiles.length}`);

function collectMarkdownFiles(directory) {
  const files = [];
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (entry.isDirectory() && ignoredDirectories.has(entry.name)) continue;
    const path = join(directory, entry.name);
    if (entry.isDirectory()) files.push(...collectMarkdownFiles(path));
    else if (entry.isFile() && extname(entry.name).toLowerCase() === ".md") files.push(path);
  }
  return files;
}

function markdownTargets(content) {
  const targets = [];
  const pattern = /!?(?:\[[^\]]*\])\((<[^>]+>|[^\s)]+)(?:\s+["'][^"']*["'])?\)/g;
  for (const match of content.matchAll(pattern)) {
    targets.push(match[1].replace(/^<|>$/g, ""));
  }
  return targets;
}

function decodeTarget(target) {
  try {
    return decodeURIComponent(target.trim());
  } catch {
    return target.trim();
  }
}

function isExternal(target) {
  return /^(?:[a-z][a-z0-9+.-]*:|\/\/)/i.test(target) || target.startsWith("/");
}

function existsAsDocumentTarget(path) {
  return existsSync(path);
}

function relative(path) {
  return normalize(path.slice(root.length + 1));
}
