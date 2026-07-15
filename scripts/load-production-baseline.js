const baseURL = (process.env.AREAFLOW_LOAD_URL ?? "http://127.0.0.1:3847").replace(/\/$/, "");
const sustainedRPS = numberEnv("AREAFLOW_LOAD_SUSTAINED_RPS", 50);
const sustainedSeconds = numberEnv("AREAFLOW_LOAD_SUSTAINED_SECONDS", 10);
const peakRPS = numberEnv("AREAFLOW_LOAD_PEAK_RPS", 200);
const peakSeconds = numberEnv("AREAFLOW_LOAD_PEAK_SECONDS", 2);
const concurrency = numberEnv("AREAFLOW_LOAD_CONCURRENCY", 100);
const maxP95Ms = numberEnv("AREAFLOW_LOAD_MAX_P95_MS", 500);
const writeProject = process.env.AREAFLOW_LOAD_WRITE_PROJECT ?? "";
const writeRequests = numberEnv("AREAFLOW_LOAD_WRITE_REQUESTS", 20);
const maxWriteP95Ms = numberEnv("AREAFLOW_LOAD_MAX_WRITE_P95_MS", 1000);

async function request(path = "/api/v1/health") {
  const started = performance.now();
  try {
    const response = await fetch(`${baseURL}${path}`, { signal: AbortSignal.timeout(5000) });
    await response.arrayBuffer();
    return { ok: response.ok, latency: performance.now() - started };
  } catch {
    return { ok: false, latency: performance.now() - started };
  }
}

async function concurrencyPhase() {
  const results = await Promise.all(Array.from({ length: concurrency }, () => request("/api/v1/ready")));
  return summarize("concurrency", results, { concurrency });
}

async function ratePhase(name, rps, seconds) {
  const results = [];
  const pending = [];
  const started = performance.now();
  const total = rps * seconds;
  for (let index = 0; index < total; index++) {
    const target = started + (index * 1000) / rps;
    const delay = target - performance.now();
    if (delay > 0) await new Promise((resolve) => setTimeout(resolve, delay));
    pending.push(request().then((result) => results.push(result)));
  }
  await Promise.all(pending);
  return summarize(name, results, { rps, seconds });
}

async function writePhase() {
  const nonce = `${Date.now()}-${process.pid}`;
  const results = [];
  for (let index = 0; index < writeRequests; index++) {
    const started = performance.now();
    try {
      const response = await fetch(`${baseURL}/api/v1/projects/${encodeURIComponent(writeProject)}/workflow-versions`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          display_label: `capacity-${nonce}-${index}`,
          idempotency_key: `capacity-${nonce}-${index}`,
          actor: "capacity-smoke",
          reason: "controlled write latency verification",
        }),
        signal: AbortSignal.timeout(5000),
      });
      await response.arrayBuffer();
      results.push({ ok: response.ok, latency: performance.now() - started });
    } catch {
      results.push({ ok: false, latency: performance.now() - started });
    }
  }
  return summarize("controlled_write", results, { max_p95_ms: maxWriteP95Ms });
}

function summarize(name, results, details) {
  const latencies = results.map((result) => result.latency).sort((a, b) => a - b);
  const failures = results.filter((result) => !result.ok).length;
  const p95 = percentile(latencies, 0.95);
  return { name, requests: results.length, failures, p95_ms: Number(p95.toFixed(2)), ...details };
}

function percentile(values, ratio) {
  if (values.length === 0) return Infinity;
  return values[Math.min(values.length - 1, Math.ceil(values.length * ratio) - 1)];
}

function numberEnv(key, fallback) {
  const value = Number(process.env[key] ?? fallback);
  if (!Number.isFinite(value) || value <= 0) throw new Error(`${key} must be a positive number`);
  return value;
}

async function main() {
  const phases = [
    await concurrencyPhase(),
    await ratePhase("sustained", sustainedRPS, sustainedSeconds),
    await ratePhase("peak", peakRPS, peakSeconds),
  ];
	if (writeProject) phases.push(await writePhase());
  console.log(JSON.stringify({ base_url: baseURL, max_p95_ms: maxP95Ms, phases }));
  if (phases.some((phase) => phase.failures > 0 || phase.p95_ms > (phase.name === "controlled_write" ? maxWriteP95Ms : maxP95Ms))) process.exitCode = 1;
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
