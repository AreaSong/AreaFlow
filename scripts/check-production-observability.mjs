import { readFileSync } from "node:fs";

const rules = readFileSync("deploy/production/prometheus-rules.yaml", "utf8");
const metrics = readFileSync("internal/observability/metrics.go", "utf8");
const failures = [];

const requiredAlerts = [
  "AreaFlowAvailabilityBudgetBurn",
  "AreaFlowReadLatencyHigh",
  "AreaFlowWriteLatencyHigh",
  "AreaFlowDependencyUnavailable",
  "AreaFlowDatabasePoolSaturation",
  "AreaFlowDependencyOperationErrorRateHigh",
];

for (const alert of requiredAlerts) {
  if (!rules.includes(`alert: ${alert}`)) failures.push(`missing alert ${alert}`);
}

const referencedMetrics = new Set(rules.match(/areaflow_[a-z0-9_]+/g) ?? []);
for (const metric of referencedMetrics) {
  const baseMetric = metric.replace(/_bucket$/, "");
  if (!metrics.includes(`Name: "${baseMetric}"`)) {
    failures.push(`alert rule references unknown application metric ${metric}`);
  }
}

if (failures.length > 0) {
  console.error(`check-production-observability: ${failures.length} violation(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log(`check-production-observability: ok alerts=${requiredAlerts.length} metrics=${referencedMetrics.size}`);
