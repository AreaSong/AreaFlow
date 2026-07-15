import { createRequire } from "node:module";

const require = createRequire(`${process.cwd()}/package.json`);
const { chromium } = require("playwright");

const [, , baseURL, projectKey, readyWorkflowLabel = "", mode = "fixture"] = process.argv;

if (!baseURL || !projectKey || (mode === "fixture" && !readyWorkflowLabel)) {
  console.error("usage: smoke-web-check.mjs <web-url> <project-key> <ready-workflow-label> [fixture|real-areamatrix]");
  process.exit(1);
}
if (mode !== "fixture" && mode !== "real-areamatrix") {
  console.error(`unsupported smoke-web-check mode: ${mode}`);
  process.exit(1);
}

const isRealAreaMatrix = mode === "real-areamatrix";
const requiredAreaMatrixPhrase = "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json";
const seenPaths = new Set();
const nonV1APIRequests = [];
const nonGetAPIRequests = [];
const pageErrors = [];
const failedResponses = [];

const browser = await chromium.launch({ headless: true });
try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (!url.pathname.startsWith("/api")) return;
    if (!url.pathname.startsWith("/api/v1")) {
      nonV1APIRequests.push(`${request.method()} ${url.pathname}`);
      return;
    }
    seenPaths.add(url.pathname);
    if (request.method() !== "GET") nonGetAPIRequests.push(`${request.method()} ${url.pathname}`);
  });
  page.on("pageerror", (error) => pageErrors.push(error.message));
  page.on("response", (response) => {
    const url = new URL(response.url());
    if (url.pathname.startsWith("/api/v1") && response.status() >= 400) {
      failedResponses.push(`${response.status()} ${url.pathname}`);
    }
  });

  await openRoute(page, baseURL, "/", projectKey, "Overview");
  await page.waitForFunction((key) => document.body.innerText.includes(key) || document.body.innerText.includes("AreaMatrix"), projectKey);

  const routes = [
    ["/projects", "Projects"],
    ["/workflows", "Workflows"],
    ["/runs", "Runs"],
    ["/workers", "Workers"],
    ["/artifacts", "Artifacts"],
    ["/audit", "Audit"],
    ["/operations", "Operations"],
    ["/access", "Access"],
  ];
  for (const [path, heading] of routes) {
    await openRoute(page, baseURL, path, projectKey, heading);
    if (path !== "/operations" && path !== "/access") {
      await page.getByRole("combobox", { name: "Sort results" }).waitFor({ timeout: 20000 });
    }
  }

  if (isRealAreaMatrix) {
    await verifyCompatibilityWorkspace(page, baseURL, projectKey);
  } else {
    await verifyWorkflowAndRunPages(page, baseURL, projectKey, readyWorkflowLabel);
  }
  await verifyResourceDetailPages(page, baseURL, projectKey);
  await verifyNotFoundPage(page, baseURL, projectKey);

  await verifyMobileLayout(page, baseURL, projectKey);

  const requiredPaths = [
    "/api/v1/projects",
    "/api/v1/workflows",
    "/api/v1/runs",
    "/api/v1/workers",
    "/api/v1/artifacts",
    "/api/v1/audit-events",
    "/api/v1/ops/readiness",
    `/api/v1/projects/${projectKey}/role-bindings`,
  ];
  for (const path of requiredPaths) {
    if (!seenPaths.has(path)) throw new Error(`required API path was not requested: ${path}`);
  }
  if (failedResponses.length) throw new Error(`failed API responses: ${failedResponses.join(", ")}`);
  if (nonV1APIRequests.length) throw new Error(`non-v1 API requests: ${nonV1APIRequests.join(", ")}`);
  if (nonGetAPIRequests.length) throw new Error(`unexpected Web writes: ${nonGetAPIRequests.join(", ")}`);
  if (pageErrors.length) throw new Error(`page errors: ${pageErrors.join(" | ")}`);

  console.log(`smoke-web-check: ok mode=${mode} routes=10 api_paths=${seenPaths.size}`);
} finally {
  await browser.close();
}

async function openRoute(page, baseURL, path, projectKey, heading) {
  const url = new URL(path, baseURL);
  url.searchParams.set("project", projectKey);
  await page.goto(url.toString(), { waitUntil: "domcontentloaded" });
  try {
    await page.getByRole("heading", { name: heading, exact: true }).waitFor({ timeout: 20000 });
  } catch (error) {
    const body = (await page.locator("body").innerText().catch(() => "")).slice(0, 2000);
    throw new Error(`route ${path} did not render ${heading}; url=${page.url()} body=${JSON.stringify(body)} failed_responses=${failedResponses.join(",")} page_errors=${pageErrors.join("|")}`, { cause: error });
  }
  await page.waitForFunction(() => !document.body.innerText.includes("Loading "), null, { timeout: 20000 }).catch(() => undefined);
}

async function verifyWorkflowAndRunPages(page, baseURL, projectKey, readyWorkflowLabel) {
  await openRoute(page, baseURL, "/workflows", projectKey, "Workflows");
  const workflowSelect = page.getByRole("combobox", { name: "Workflow version" });
  await workflowSelect.selectOption(readyWorkflowLabel);
  await page.getByRole("heading", { name: "Stage board" }).waitFor({ timeout: 20000 });
  await page.waitForFunction((label) => document.body.innerText.includes(label), readyWorkflowLabel);

  await openRoute(page, baseURL, "/runs", projectKey, "Runs");
  const versionSelect = page.getByRole("combobox", { name: "Run workflow version" });
  await versionSelect.selectOption(readyWorkflowLabel);
  const firstRun = page.locator(".resource-list button").first();
  await firstRun.waitFor({ timeout: 20000 });
  await firstRun.click();
  await page.getByRole("heading", { name: /Run #\d+/ }).last().waitFor({ timeout: 20000 });
}

async function verifyCompatibilityWorkspace(page, baseURL, projectKey) {
  const authorizationPromise = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname === `/api/v1/projects/${projectKey}/status-projections/authorization` && response.status() === 200;
  });
  const gatePromise = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname === `/api/v1/projects/${projectKey}/status-projections/apply-gate` && response.status() === 200;
  });
  const url = new URL("/operations/compatibility", baseURL);
  url.searchParams.set("project", projectKey);
  await page.goto(url.toString(), { waitUntil: "domcontentloaded" });
  const authorization = await authorizationPromise.then((response) => response.json());
  const gate = await gatePromise.then((response) => response.json());
  if (authorization.required_authorization_phrase !== requiredAreaMatrixPhrase) {
    throw new Error("status projection authorization phrase does not match the managed project contract");
  }
  if (gate.required_authorization_phrase !== requiredAreaMatrixPhrase) {
    throw new Error("status projection gate phrase does not match the managed project contract");
  }
  await page.locator('[data-panel="status-projection-authorization"]').waitFor({ timeout: 20000 });
  await page.waitForFunction((phrase) => document.body.innerText.includes(phrase), requiredAreaMatrixPhrase);
}

async function verifyResourceDetailPages(page, baseURL, projectKey) {
  for (const resource of ["workers", "artifacts"]) {
    const heading = resource === "workers" ? "Workers" : "Artifacts";
    await openRoute(page, baseURL, `/${resource}`, projectKey, heading);
    const firstResource = page.locator(".resource-list button").first();
    await firstResource.waitFor({ timeout: 20000 });
    await firstResource.click();
    await page.waitForURL((url) => new RegExp(`/${resource}/[^/?]+`).test(url.pathname), { timeout: 20000 });
    await page.reload({ waitUntil: "domcontentloaded" });
    await page.getByRole("heading", { name: heading, exact: true }).waitFor({ timeout: 20000 });
    await page.locator(".resource-detail .content-section h2").first().waitFor({ timeout: 20000 });
  }
}

async function verifyNotFoundPage(page, baseURL, projectKey) {
  await openRoute(page, baseURL, "/route-that-does-not-exist", projectKey, "Page not found");
  const url = new URL(page.url());
  if (url.pathname !== "/route-that-does-not-exist") {
    throw new Error(`unknown route redirected unexpectedly: ${url.pathname}`);
  }
}

async function verifyMobileLayout(page, baseURL, projectKey) {
  await page.setViewportSize({ width: 390, height: 844 });
  await openRoute(page, baseURL, "/runs", projectKey, "Runs");
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  if (overflow > 1) throw new Error(`mobile layout has ${overflow}px horizontal overflow`);
  const navLabels = await page.locator(".primary-nav a span").allTextContents();
  if (navLabels.length !== 9 || navLabels.some((label) => !label.trim())) {
    throw new Error(`mobile navigation labels are incomplete: ${navLabels.join(", ")}`);
  }
}
