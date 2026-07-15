import http from "node:http";

const listenPort = Number(process.env.AREAFLOW_HA_PROXY_PORT ?? 3860);
const backends = (process.env.AREAFLOW_HA_BACKENDS ?? "http://127.0.0.1:3857,http://127.0.0.1:3858")
  .split(",").map((value) => value.trim()).filter(Boolean);
let cursor = 0;

async function healthy(baseURL) {
  try {
    const response = await fetch(`${baseURL}/api/v1/ready`, { signal: AbortSignal.timeout(500) });
    return response.ok;
  } catch {
    return false;
  }
}

async function selectBackend() {
  for (let offset = 0; offset < backends.length; offset++) {
    const index = (cursor + offset) % backends.length;
    if (await healthy(backends[index])) {
      cursor = (index + 1) % backends.length;
      return backends[index];
    }
  }
  return "";
}

const server = http.createServer(async (request, response) => {
  const backend = await selectBackend();
  if (!backend) {
    response.writeHead(503, { "content-type": "application/json" });
    response.end('{"status":"unavailable"}');
    return;
  }
  const upstream = http.request(new URL(request.url, backend), {
    method: request.method,
    headers: { ...request.headers, host: new URL(backend).host },
  }, (upstreamResponse) => {
    response.writeHead(upstreamResponse.statusCode ?? 502, upstreamResponse.headers);
    upstreamResponse.pipe(response);
  });
  upstream.on("error", async () => {
    const retryBackend = await selectBackend();
    if (!retryBackend) {
      response.writeHead(503, { "content-type": "application/json" });
      response.end('{"status":"unavailable"}');
      return;
    }
    const retry = http.request(new URL(request.url, retryBackend), { method: request.method, headers: request.headers }, (retryResponse) => {
      response.writeHead(retryResponse.statusCode ?? 502, retryResponse.headers);
      retryResponse.pipe(response);
    });
    retry.on("error", () => response.destroy());
    retry.end();
  });
  request.pipe(upstream);
});

server.listen(listenPort, "127.0.0.1");
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => server.close(() => process.exit(0)));
