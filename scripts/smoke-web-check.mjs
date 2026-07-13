import { createRequire } from "node:module";

const require = createRequire(`${process.cwd()}/package.json`);
const { chromium } = require("playwright");

const [, , url, projectKey, readyWorkflowLabel = "", mode = "fixture"] = process.argv;

if (!url || !projectKey || (mode !== "real-areamatrix" && !readyWorkflowLabel)) {
  console.error("usage: smoke-web-check.mjs <web-url> <project-key> <ready-workflow-label> [fixture|real-areamatrix]");
  process.exit(1);
}

if (mode !== "fixture" && mode !== "real-areamatrix") {
  console.error(`unsupported smoke-web-check mode: ${mode}`);
  process.exit(1);
}

const isRealAreaMatrix = mode === "real-areamatrix";
const requiredAreaMatrixPhrase = "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json";

const expectedPaths = new Set([
  "/api/v1/projects",
  `/api/v1/projects/${projectKey}/summary`,
  `/api/v1/projects/${projectKey}/readiness`,
  `/api/v1/projects/${projectKey}/workflow-versions`,
  `/api/v1/projects/${projectKey}/workers`,
  `/api/v1/projects/${projectKey}/shim-authorization`,
  `/api/v1/projects/${projectKey}/shim-apply-packet`,
  `/api/v1/projects/${projectKey}/shim-apply-gate`,
  `/api/v1/projects/${projectKey}/status-projections/authorization`,
  `/api/v1/projects/${projectKey}/status-projections/apply-packet`,
  `/api/v1/projects/${projectKey}/status-projections/apply-gate`,
  `/api/v1/projects/${projectKey}/execution-cutover-readiness`,
  `/api/v1/projects/${projectKey}/execution-forwarding-v1-readiness`,
  `/api/v1/projects/${projectKey}/execution-forwarding-v1-apply-preview`,
  `/api/v1/projects/${projectKey}/execution-forwarding-v1-apply-packet`,
  `/api/v1/projects/${projectKey}/execution-forwarding-v1-command-preview`,
  `/api/v1/projects/${projectKey}/execution-forwarding-v1-rollback-preview`,
  `/api/v1/projects/${projectKey}/events`,
  `/api/v1/projects/${projectKey}/events/stream`,
  "/api/v1/worker-pool/summary",
  "/api/v1/worker-pool/schedule-preview",
  "/api/v1/web/write-action-gate",
  "/api/v1/completion-audit/snapshot-readiness",
  "/api/v1/ops/readiness",
  "/api/v1/release/final-gate",
  "/api/v1/release/evidence-bundle",
  "/api/v1/release/package-preview",
  "/api/v1/release/distribution-preview",
  "/api/v1/release/publish-gate",
  "/api/v1/release/publish-approval-preview",
  "/api/v1/release/rollout-plan-preview",
]);

const seenPaths = new Set();
const nonV1APIRequests = [];
const nonGetAPIRequests = [];
const pageErrors = [];
const failedResponses = [];
const runDetailRequests = [];

const browser = await launchBrowser();
try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1200 } });
  page.on("request", (request) => {
    const nextURL = new URL(request.url());
    if (!nextURL.pathname.startsWith("/api")) {
      return;
    }
    if (!nextURL.pathname.startsWith("/api/v1")) {
      nonV1APIRequests.push(`${request.method()} ${nextURL.pathname}`);
      return;
    }
    seenPaths.add(nextURL.pathname);
    if (/^\/api\/v1\/runs\/\d+$/.test(nextURL.pathname)) {
      runDetailRequests.push(`${nextURL.pathname}?${nextURL.searchParams.toString()}`);
    }
    if (request.method() !== "GET") {
      nonGetAPIRequests.push(`${request.method()} ${nextURL.pathname}`);
    }
  });
  page.on("pageerror", (error) => {
    pageErrors.push(error.message);
  });
  page.on("response", (response) => {
    const nextURL = new URL(response.url());
    if (nextURL.pathname.startsWith("/api/v1") && response.status() >= 400) {
      if (
        response.status() === 404 &&
        nextURL.pathname.startsWith(`/api/v1/projects/${projectKey}/workflow-versions/`) &&
        !nextURL.pathname.includes(`/workflow-versions/${readyWorkflowLabel}/`)
      ) {
        return;
      }
      failedResponses.push(`${response.status()} ${nextURL.pathname}`);
    }
  });

  let statusAuthorization = null;
  let statusApplyPacket = null;
  let statusApplyGate = null;

  const projectListResponsePromise = page.waitForResponse((response) => {
    const nextURL = new URL(response.url());
    return nextURL.pathname === "/api/v1/projects" && response.status() === 200;
  });
  const statusAuthorizationResponsePromise = isRealAreaMatrix
    ? page.waitForResponse((response) => {
        const nextURL = new URL(response.url());
        return (
          nextURL.pathname === `/api/v1/projects/${projectKey}/status-projections/authorization` &&
          response.status() === 200
        );
      })
    : null;
  const statusApplyPacketResponsePromise = isRealAreaMatrix
    ? page.waitForResponse((response) => {
        const nextURL = new URL(response.url());
        return (
          nextURL.pathname === `/api/v1/projects/${projectKey}/status-projections/apply-packet` &&
          response.status() === 200
        );
      })
    : null;
  const statusApplyGateResponsePromise = isRealAreaMatrix
    ? page.waitForResponse((response) => {
        const nextURL = new URL(response.url());
        return (
          nextURL.pathname === `/api/v1/projects/${projectKey}/status-projections/apply-gate` &&
          response.status() === 200
        );
      })
    : null;

  await page.goto(url, { waitUntil: "domcontentloaded" });
  const projectListResponse = await projectListResponsePromise;
  const projectList = await projectListResponse.json();
  const selectedProject = projectList.projects.find((project) => project.key === projectKey);
  if (!selectedProject) {
    throw new Error(`project ${projectKey} was not returned by /api/v1/projects`);
  }
  await page.waitForFunction(() => document.body.innerText.includes("AreaMatrix"), null, {
    timeout: 20000,
  });
  await clickProjectButton(page, selectedProject.key, selectedProject.name || selectedProject.key);

  if (isRealAreaMatrix) {
    statusAuthorization = await statusAuthorizationResponsePromise.then((response) => response.json());
    statusApplyPacket = await statusApplyPacketResponsePromise.then((response) => response.json());
    statusApplyGate = await statusApplyGateResponsePromise.then((response) => response.json());
    assertRealAreaMatrixStatusProjection(statusAuthorization, statusApplyPacket, statusApplyGate);
    const authorizationPanel = page.locator('[data-panel="status-projection-authorization"]');
    const gatePanel = page.locator('[data-panel="status-projection-gate"]');
    try {
      await authorizationPanel.scrollIntoViewIfNeeded({ timeout: 20000 });
      await gatePanel.scrollIntoViewIfNeeded({ timeout: 20000 });
      await page.waitForFunction(
        (phrase) => {
          const authorization = document.querySelector('[data-panel="status-projection-authorization"]')?.textContent ?? "";
          const gate = document.querySelector('[data-panel="status-projection-gate"]')?.textContent ?? "";
          return (
            authorization.includes("Status Projection") &&
            authorization.includes("Authorization Preview") &&
            authorization.includes("scope=package_a_status_projection_preflight_only") &&
            authorization.includes("not_real_100=true") &&
            authorization.includes(`required=${phrase}`) &&
            authorization.includes("apply_open=false") &&
            authorization.includes("project_write=false") &&
            authorization.includes("execution_write=false") &&
            authorization.includes("engine_call=false") &&
            gate.includes("Package A Gate") &&
            gate.includes("scope=package_a_status_projection_preflight_only") &&
            gate.includes("not_real_100=true") &&
            gate.includes(`required=${phrase}`) &&
            gate.includes("eligible_is_not_apply=true") &&
            gate.includes("separate_apply=true") &&
            gate.includes("status_projection=false") &&
            gate.includes("project_write=false") &&
            gate.includes("execution_write=false") &&
            gate.includes("engine_call=false")
          );
        },
        requiredAreaMatrixPhrase,
        { timeout: 20000 },
      );
    } catch (error) {
      const text = await page.evaluate(() => document.body.textContent ?? "");
      const authorizationText = await authorizationPanel.textContent().catch(() => "");
      const gateText = await gatePanel.textContent().catch(() => "");
      throw new Error(
        `real AreaMatrix status projection UI did not render required phrase: ${error.message}; failed API responses: ${failedResponses.join(", ") || "none"}; authorization panel:\n${(authorizationText ?? "").slice(0, 2000)}\nPackage A panel:\n${(gateText ?? "").slice(0, 2000)}\nvisible text:\n${text.slice(0, 4000)}`,
      );
    }
  } else {
    await page.waitForFunction(
      (label) =>
        Array.from(document.querySelectorAll('select[aria-label="Workflow version"] option')).some(
          (option) => option.value === label,
        ),
      readyWorkflowLabel,
      { timeout: 20000 },
    );
    await page.selectOption('select[aria-label="Workflow version"]', readyWorkflowLabel);
    await page.waitForFunction(
      (label) =>
        document.body.innerText.includes(label) &&
        document.body.innerText.includes("runner_preview") &&
        document.body.innerText.includes("worker_run_once_report"),
      readyWorkflowLabel,
      { timeout: 20000 },
    );
    const runDetailResponse = page.waitForResponse((response) => {
      const nextURL = new URL(response.url());
      return (
        /^\/api\/v1\/runs\/\d+$/.test(nextURL.pathname) &&
        nextURL.searchParams.get("project_key") === projectKey &&
        response.status() === 200
      );
    });
    await page.locator("button.record").filter({ hasText: "runner_preview" }).first().click();
    await runDetailResponse;
  }

  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await page.waitForFunction(
    (key) =>
      document.body.innerText.includes("Schedule Preview") &&
      document.body.innerText.includes("Operations") &&
      document.body.innerText.includes("Readiness") &&
      document.body.innerText.includes("Web Action Gate") &&
      document.body.innerText.includes("Shim Authorization") &&
      document.body.innerText.includes("Status Projection") &&
      document.body.innerText.includes("Execution Cutover") &&
      document.body.innerText.includes("Forwarding v1") &&
      document.body.innerText.includes("Packet Gate") &&
      document.body.innerText.includes("Command Preview") &&
      document.body.innerText.includes("Completion Snapshot") &&
      document.body.innerText.includes("Release Final Gate") &&
      document.body.innerText.includes("Release Evidence") &&
      document.body.innerText.includes("Release Package") &&
      document.body.innerText.includes("Release Distribution") &&
      document.body.innerText.includes("Release Publish") &&
      document.body.innerText.includes("Release Approval") &&
      document.body.innerText.includes("Release Rollout") &&
      document.body.innerText.includes("Worker Pool") &&
      document.body.innerText.includes(key),
    projectKey,
    { timeout: 10000 },
  ).catch(() => undefined);
  await page.waitForFunction(
    () => document.body.innerText.includes("local_worker") || document.body.innerText.includes("engine_profile_disabled"),
    null,
    { timeout: 10000 },
  ).catch(() => undefined);

  if (failedResponses.length > 0) {
    throw new Error(`failed API responses: ${failedResponses.join(", ")}`);
  }
  if (nonGetAPIRequests.length > 0) {
    throw new Error(`dashboard issued non-GET API requests: ${nonGetAPIRequests.join(", ")}`);
  }
  if (nonV1APIRequests.length > 0) {
    throw new Error(`dashboard issued non-v1 API requests: ${nonV1APIRequests.join(", ")}`);
  }
  if (pageErrors.length > 0) {
    throw new Error(`page errors: ${pageErrors.join(" | ")}`);
  }
  if (!isRealAreaMatrix && !runDetailRequests.some((request) => request.includes(`project_key=${encodeURIComponent(projectKey)}`))) {
    throw new Error(`run detail request did not include project_key: ${runDetailRequests.join(", ")}`);
  }
  if (seenPaths.has(`/api/v1/projects/${projectKey}/status-projections/apply`)) {
    throw new Error("dashboard requested status projection apply endpoint");
  }

  const text = await page.evaluate(() => document.body.textContent ?? "");
  if (isRealAreaMatrix) {
    assertRealAreaMatrixStatusProjection(statusAuthorization, statusApplyPacket, statusApplyGate);
    assertRealAreaMatrixText(text);
  } else {
    assertFixtureText(text);
  }

  const missingPaths = [...expectedPaths].filter((path) => !seenPaths.has(path));
  if (missingPaths.length > 0) {
    throw new Error(`missing expected API calls: ${missingPaths.join(", ")}`);
  }
} finally {
  await browser.close();
}

async function launchBrowser() {
  try {
    return await chromium.launch();
  } catch (error) {
    try {
      return await chromium.launch({ channel: "chrome" });
    } catch (fallbackError) {
      throw new Error(
        `launch chromium failed: ${error.message}; chrome channel fallback failed: ${fallbackError.message}`,
      );
    }
  }
}

async function clickProjectButton(page, projectKey, projectName) {
  const keyedButton = page.locator(`button.project-button[data-project-key="${cssAttributeValue(projectKey)}"]`);
  try {
    await keyedButton.first().waitFor({ state: "visible", timeout: 20000 });
    await keyedButton.first().click();
    return;
  } catch {
    // Fall through to the legacy text lookup for older dashboard builds.
  }

  const buttons = page.locator("button.project-button");
  const count = await buttons.count();
  for (let index = 0; index < count; index += 1) {
    const button = buttons.nth(index);
    const text = ((await button.textContent()) ?? "").replace(/\s+/g, " ").trim();
    if (text.includes(projectName)) {
      await button.click();
      return;
    }
  }
  throw new Error(`project button not found for ${projectName}`);
}

function cssAttributeValue(value) {
  return String(value).replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

function assertRealAreaMatrixStatusProjection(authorization, applyPacket, applyGate) {
  const protectedPathFingerprint = authorization.protected_path_fingerprint_sha256 ?? "";
  assertSHA256(protectedPathFingerprint, "authorization protected path fingerprint");
  assertEqual(authorization.required_authorization_phrase, requiredAreaMatrixPhrase, "authorization phrase");
  assertEqual(applyPacket.required_authorization_phrase, requiredAreaMatrixPhrase, "packet phrase");
  assertEqual(applyGate.required_authorization_phrase, requiredAreaMatrixPhrase, "gate phrase");
  assertEqual(authorization.mode, "status_projection_apply_authorization_preview_v1", "authorization mode");
  assertEqual(applyPacket.mode, "status_projection_apply_packet_preview_v1", "packet mode");
  assertEqual(authorization.claim_scope, "package_a_status_projection_preflight_only", "authorization claim scope");
  assertEqual(applyPacket.claim_scope, "package_a_status_projection_preflight_only", "packet claim scope");
  assertEqual(applyGate.claim_scope, "package_a_status_projection_preflight_only", "gate claim scope");
  assertEqual(authorization.not_real_100, true, "authorization not real 100");
  assertEqual(applyPacket.not_real_100, true, "packet not real 100");
  assertEqual(applyGate.not_real_100, true, "gate not real 100");
  assertEqual(authorization.apply_open, false, "authorization apply_open");
  assertEqual(applyPacket.gate.apply_command_eligible, false, "packet gate eligible");
  assertEqual(applyGate.apply_command_eligible, false, "gate eligible");
  assertEqual(applyPacket.apply_command_eligible_is_not_apply, true, "packet eligible is not apply");
  assertEqual(applyPacket.requires_separate_apply_command, true, "packet requires separate apply");
  assertEqual(applyPacket.gate.apply_command_eligible_is_not_apply, true, "packet gate eligible is not apply");
  assertEqual(applyPacket.gate.requires_separate_apply_command, true, "packet gate requires separate apply");
  assertEqual(applyGate.apply_command_eligible_is_not_apply, true, "gate eligible is not apply");
  assertEqual(applyGate.requires_separate_apply_command, true, "gate requires separate apply");
  assertEqual(authorization.project_write_attempted, false, "authorization project write");
  assertEqual(authorization.execution_write_attempted, false, "authorization execution write");
  assertEqual(authorization.engine_call_attempted, false, "authorization engine call");
  assertEqual(applyPacket.project_write_attempted, false, "packet project write");
  assertEqual(applyPacket.execution_write_attempted, false, "packet execution write");
  assertEqual(applyPacket.engine_call_attempted, false, "packet engine call");
  assertEqual(applyGate.project_write_attempted, false, "gate project write");
  assertEqual(applyGate.execution_write_attempted, false, "gate execution write");
  assertEqual(applyGate.engine_call_attempted, false, "gate engine call");
  assertEqual(applyGate.status_projection_written, false, "gate status projection write");
  assertEqual(
    applyPacket.packet?.protected_path_fingerprint_sha256,
    protectedPathFingerprint,
    "packet protected path fingerprint",
  );
  assertEqual(
    applyPacket.api_request?.protected_path_fingerprint_sha256,
    protectedPathFingerprint,
    "api request protected path fingerprint",
  );
  assertGateFingerprintItem(applyPacket.gate, protectedPathFingerprint, "packet gate", true);
  assertGateFingerprintItem(applyGate, protectedPathFingerprint, "standalone gate", false);
}

function assertGateFingerprintItem(gate, protectedPathFingerprint, label, requireActualMatch) {
  const protectedPathGateItem = gate.items?.find((item) => item.key === "protected_path_fingerprint_sha256");
  if (!protectedPathGateItem) {
    throw new Error(`${label} did not include protected_path_fingerprint_sha256 item`);
  }
  if (requireActualMatch || protectedPathGateItem.actual) {
    assertEqual(protectedPathGateItem.actual, protectedPathFingerprint, `${label} protected path fingerprint actual`);
  }
  assertEqual(protectedPathGateItem.expected, protectedPathFingerprint, `${label} protected path fingerprint expected`);
}

function assertRealAreaMatrixText(text) {
  assertIncludes(text, "AreaFlow");
  assertIncludes(text, "Workflow control plane");
  assertIncludes(text, "AreaMatrix");
  assertReal100GuardrailText(text);
  assertIncludes(text, "Status Projection");
  assertIncludes(text, "Authorization Preview");
  assertIncludes(text, "Package A Gate");
  assertIncludes(text, "status_projection_apply_authorization_preview_v1");
  assertIncludes(text, "status_projection_apply_packet_preview_v1");
  assertIncludes(text, "target=.areaflow/status.json");
  assertIncludes(text, "preview_only=true");
  assertIncludes(text, `required=${requiredAreaMatrixPhrase}`);
  assertIncludes(text, "apply_open=false");
  assertIncludes(text, "apply_run=false");
  assertIncludes(text, "applied=false");
  assertIncludes(text, "project_write=false");
  assertIncludes(text, "execution_write=false");
  assertIncludes(text, "engine_call=false");
  assertIncludes(text, "status_projection=false");
}

function assertFixtureText(text) {
  assertIncludes(text, "AreaFlow");
  assertIncludes(text, "Workflow control plane");
  assertIncludes(text, "AreaMatrix Web Fixture");
  assertReal100GuardrailText(text);
  assertIncludes(text, "Version Timeline");
  assertIncludes(text, "Stage Board");
  assertIncludes(text, "Run Timeline");
  assertIncludes(text, "Version Files");
  assertIncludes(text, "Approval Records");
  assertIncludes(text, "Worker Status");
  assertIncludes(text, "Worker Pool");
  assertIncludes(text, "Schedule Preview");
  assertIncludes(text, "Operations");
  assertIncludes(text, "Readiness");
  assertIncludes(text, "Web Action Gate");
  assertIncludes(text, "Shim Authorization");
  assertIncludes(text, "AreaMatrix Shim");
  assertIncludes(text, "required preflight");
  assertIncludes(text, "post-edit verification");
  assertIncludes(text, "rollback scope");
  assertIncludes(text, "preflight=");
  assertIncludes(text, "post_edit=");
  assertIncludes(text, "rollback=");
  assertIncludes(text, "Execution Cutover");
  assertIncludes(text, "AreaMatrix Readiness");
  assertIncludes(text, "Forwarding v1");
  assertIncludes(text, "Read-only Scope");
  assertIncludes(text, "Apply Preview");
  assertIncludes(text, "Packet Gate");
  assertIncludes(text, "legacy_ref=");
  assertIncludes(text, "fingerprint_ref=");
  assertIncludes(text, "Command Preview");
  assertIncludes(text, "Rollback Preview");
  assertIncludes(text, "would_forward_after_approval");
  assertIncludes(text, "blocked_task_type_fail_closed");
  assertIncludes(text, "targets=");
  assertIncludes(text, "blocked=");
  assertIncludes(text, "Completion Snapshot");
  assertIncludes(text, "Release Candidate Gate");
  assertIncludes(text, "Release Final Gate");
  assertIncludes(text, "Release Readiness");
  assertIncludes(text, "Release Evidence");
  assertIncludes(text, "Evidence Bundle");
  assertIncludes(text, "Release Package");
  assertIncludes(text, "Package Preview");
  assertIncludes(text, "Release Distribution");
  assertIncludes(text, "Distribution Preview");
  assertIncludes(text, "Release Publish");
  assertIncludes(text, "Publish Gate");
  assertIncludes(text, "Release Approval");
  assertIncludes(text, "Publish Approval");
  assertIncludes(text, "Release Rollout");
  assertIncludes(text, "Rollout Plan");
  assertIncludes(text, "Disabled Writes");
  assertIncludes(text, readyWorkflowLabel);
  assertIncludes(text, "runner_preview");
  assertIncludes(text, "dry-run");
  assertIncludes(text, "runner_preview_report");
  assertIncludes(text, "worker_run_once_report");
  assertIncludes(text, projectKey);
  assertIncludes(text, "local_worker");
  assertIncludes(text, "codex-cli blocked");
  assertIncludes(text, "engine_profile_disabled");
  assertIncludes(text, "generated_write_apply_beta");
  assertIncludes(text, "read_only_operations_readiness");
  assertIncludes(text, "install_migrate_start_register_smoke");
  assertIncludes(text, "metadata_only_support_bundle_preview");
  assertIncludes(text, "migration_ledger_readiness");
  assertIncludes(text, "support_export=deferred_v1x");
  assertIncludes(text, "telemetry=local_only");
  assertIncludes(text, "db_write=false");
  assertIncludes(text, "task_loop_run=false");
  assertIncludes(text, "Shim Apply Review");
  assertIncludes(text, "Packet Gate");
  assertIncludes(text, "project.shim.apply");
  assertIncludes(text, "shim_readiness_still_blocked");
  assertIncludes(text, "area_matrix_files=false");
  assertIncludes(text, "Status Projection");
  assertIncludes(text, "Authorization Preview");
  assertIncludes(text, "Package A Gate");
  assertIncludes(text, "status_projection_apply_authorization_preview_v1");
  assertIncludes(text, "status_projection_apply_packet_preview_v1");
  assertIncludes(text, "target=.areaflow/status.json");
  assertIncludes(text, "preview_only=true");
  assertIncludes(text, "command_after_approval=");
  assertIncludes(text, "apply_open=false");
  assertIncludes(text, "apply_run=false");
  assertIncludes(text, "applied=false");
  assertIncludes(text, "required=none");
  assertIncludes(text, "eligible=");
  assertIncludes(text, "approval=");
  assertIncludes(text, "status_projection=false");
  assertIncludes(text, "explicit_execution_cutover_approval");
  assertIncludes(text, "execution_cutover_apply=false");
  assertIncludes(text, "task_loop_run_forwarded=false");
  assertIncludes(text, "read_only_shim");
  assertIncludes(text, "forwarding_command_api");
  assertIncludes(text, "rollback_to_read_only_shim");
  assertIncludes(text, "forwarding_apply=false");
  assertIncludes(text, "rollback_open=false");
  assertIncludes(text, "completion_audit_snapshot_project_mismatch");
  assertIncludes(text, "latest_class=none");
  assertIncludes(text, "has_snapshot=false");
  assertIncludes(text, "smoke_run=false");
  assertIncludes(text, "read_only_release_final_gate");
  assertIncludes(text, "read_only_release_evidence_bundle");
  assertIncludes(text, "read_only_release_package_preview");
  assertIncludes(text, "read_only_release_distribution_preview");
  assertIncludes(text, "read_only_release_publish_gate");
  assertIncludes(text, "read_only_release_publish_approval_preview");
  assertIncludes(text, "read_only_release_rollout_plan_preview");
  assertIncludes(text, "final_gate:release_readiness");
  assertIncludes(text, "evidence:release_final_gate");
  assertIncludes(text, "package:evidence:release_final_gate");
  assertIncludes(text, "distribution:git_release");
  assertIncludes(text, "publish_gate:distribution_preview");
  assertIncludes(text, "publish_approval:publish_gate");
  assertIncludes(text, "rollout_plan:publish_approval");
  assertIncludes(text, "create_release_package");
  assertIncludes(text, "compress_artifacts");
  assertIncludes(text, "create_git_tag");
  assertIncludes(text, "approve_release");
  assertIncludes(text, "create_rollout");
  assertIncludes(text, "publish_release");
  assertIncludes(text, "apply_release");
  assertIncludes(text, "scripts/task_loop/console.py");
  assertIncludes(text, "idle");
}

function assertReal100GuardrailText(text) {
  assertIncludes(text, "real_100=blocked");
  assertIncludes(text, "claim_scope=areaflow_release_preview_only");
  assertIncludes(text, "claim_scope=completion_audit_evidence_only");
  assertIncludes(text, "not_real_100=true");
  assertIncludes(text, "evidence_only=true");
  assertIncludes(text, "status_alone_is_not_completion=true");
  assertIncludes(text, "release_candidate=not_release_candidate_evidence");
  assertIncludes(text, "release_candidate=requires_release_candidate_snapshot");
  assertIncludes(text, "scope=areaflow_release_preview_only");
  assertIncludes(text, "scope=completion_audit_evidence_only");
  assertIncludes(text, "blockers=package_a_status_projection_apply_provenance_missing,real_areamatrix_read_only_shim_not_landed,+3");
  assertIncludes(text, "blockers=package_a_status_projection_apply_provenance_missing,real_areamatrix_read_only_shim_not_landed,+4");
}

function assertIncludes(text, pattern) {
  if (!text.includes(pattern)) {
    throw new Error(
      `expected page text to include ${JSON.stringify(pattern)}; visible text was:\n${text.slice(0, 4000)}`,
    );
  }
}

function assertEqual(actual, expected, label) {
  if (actual !== expected) {
    throw new Error(`${label} expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
}

function assertSHA256(value, label) {
  if (!/^[a-f0-9]{64}$/.test(value)) {
    throw new Error(`${label} expected sha256 hex, got ${JSON.stringify(value)}`);
  }
}
