import { readFileSync } from "node:fs";

const source = readFileSync("internal/api/server.go", "utf8");
const apiReference = readFileSync("docs/reference/api.md", "utf8");
const spec = JSON.parse(readFileSync("docs/reference/openapi.yaml", "utf8"));
const paths = new Set(Object.keys(spec.paths ?? {}));
const families = new Set(spec["x-areaflow-route-families"] ?? []);
const failures = [];
const routePattern = /mux\.HandleFunc\("(\/api\/[^"\n]+)"/g;

for (const match of source.matchAll(routePattern)) {
  const route = match[1];
  if (route.endsWith("/")) {
    const family = route.split("/").filter(Boolean).at(-1);
    if (!families.has(family)) failures.push(`missing documented route family: ${route}`);
    continue;
  }
  const canonical = route.replace(/^\/api/, "/api/v1");
  if (!paths.has(canonical)) failures.push(`missing OpenAPI path: ${canonical}`);
}

const documentedRoutes = new Map();
for (const line of apiReference.split("\n")) {
  const match = line.match(/^\s*(GET|POST|DELETE|GET\|POST\|DELETE)\s+(\/[^\s?]+)/);
  if (!match) continue;
  const path = match[2].startsWith("/api/v1") ? match[2] : `/api/v1${match[2]}`;
  const methods = documentedRoutes.get(path) ?? new Set();
  for (const method of match[1].toLowerCase().split("|")) methods.add(method);
  documentedRoutes.set(path, methods);
}
for (const [path, methods] of documentedRoutes) {
  const pathItem = resolvePathItem(spec, spec.paths?.[path]);
  if (!pathItem) {
    failures.push(`documented API path missing from OpenAPI: ${path}`);
    continue;
  }
  for (const method of methods) {
    if (!pathItem[method]) failures.push(`documented API method missing from OpenAPI: ${method.toUpperCase()} ${path}`);
  }
}

if (spec.openapi !== "3.1.0" || spec.info?.version !== "1.0.0") failures.push("OpenAPI and product version must be 3.1.0 / 1.0.0");
if (failures.length > 0) {
  console.error(`check-openapi-contract: ${failures.length} violation(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}
console.log(`check-openapi-contract: ok paths=${paths.size} documented=${documentedRoutes.size} families=${families.size}`);

function resolvePathItem(spec, item) {
  if (!item) return null;
  const prefix = "#/components/pathItems/";
  if (typeof item.$ref === "string" && item.$ref.startsWith(prefix)) {
    return spec.components?.pathItems?.[item.$ref.slice(prefix.length)] ?? null;
  }
  return item;
}
