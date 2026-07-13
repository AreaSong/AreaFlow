package project

import (
	"testing"
	"time"
)

func TestBuildGeneratedWriteApplyBetaGateBlocksWhenReadinessNotReady(t *testing.T) {
	generated := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	readiness := GeneratedWriteReadiness{
		Project:                   Record{ID: 1, Key: "areamatrix"},
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_readiness",
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
		ReadyForReview:            false,
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		ReviewBlockers:            []string{"required_capabilities: missing write_generated"},
		GeneratedAt:               generated,
	}

	gate := BuildGeneratedWriteApplyBetaGate(readiness, GeneratedWriteApplyBetaGateOptions{GeneratedAt: generated})

	if gate.Status != "blocked" || gate.Mode != "read_only_generated_write_apply_beta_gate" {
		t.Fatalf("unexpected beta gate: %+v", gate)
	}
	if gate.Readiness.ReadyForReview {
		t.Fatalf("nested readiness should remain not ready: %+v", gate.Readiness)
	}
	assertGeneratedApplyBetaItem(t, gate, "generated_apply_beta:readiness", "blocked")
	assertGeneratedApplyBetaItem(t, gate, "generated_apply_beta:explicit_approval", "blocked")
	if gate.ApplyOpen || gate.RealAreaMatrixWriteOpened || !gate.ApprovalRequired || gate.ApprovalStatus != "needs_approval" {
		t.Fatalf("beta gate should require approval and keep apply closed: %+v", gate)
	}
	if gate.ProjectReadAttempted || gate.ProjectWriteAttempted || gate.ExecutionWriteAttempted ||
		gate.AreaFlowArtifactWritten || gate.AreaFlowExecutionStateWritten || gate.EngineCallAttempted ||
		gate.CommandsRun || gate.SecretsResolved || gate.NetworkUsed || gate.TaskClaimed || gate.WorkerStarted ||
		gate.LeaseCreated || gate.AttemptCreated || gate.ArtifactCreated {
		t.Fatalf("beta gate should be read-only: %+v", gate)
	}
}

func TestBuildGeneratedWriteApplyBetaGateNeedsApprovalWhenReadinessReady(t *testing.T) {
	generated := time.Date(2026, 7, 2, 12, 10, 0, 0, time.UTC)
	readiness := GeneratedWriteReadiness{
		Project:                   Record{ID: 1, Key: "areamatrix"},
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_readiness",
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
		ReadyForReview:            true,
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		GeneratedOnly:             true,
		GeneratedAt:               generated,
	}

	gate := BuildGeneratedWriteApplyBetaGate(readiness, GeneratedWriteApplyBetaGateOptions{GeneratedAt: generated})

	if gate.Status != "blocked" {
		t.Fatalf("gate should remain blocked until explicit approval opens beta apply: %+v", gate)
	}
	if gate.Readiness.Project.Key != "areamatrix" || !gate.Readiness.ReadyForReview {
		t.Fatalf("nested readiness not preserved: %+v", gate.Readiness)
	}
	assertGeneratedApplyBetaItem(t, gate, "generated_apply_beta:readiness", "pass")
	approvalItem := generatedApplyBetaItemByKey(gate.Items, "generated_apply_beta:explicit_approval")
	if approvalItem.Status != "blocked" || approvalItem.ApprovalStatus != "needs_approval" {
		t.Fatalf("expected explicit approval blocker: %+v", approvalItem)
	}
	if !containsString(approvalItem.RequiredEvidence, "explicit R3 approval for real AreaMatrix generated-only apply beta") {
		t.Fatalf("approval item missing R3 evidence: %+v", approvalItem.RequiredEvidence)
	}
	scopeItem := generatedApplyBetaItemByKey(gate.Items, "generated_apply_beta:scope")
	if scopeItem.Status != "pass" || scopeItem.Metadata["allowed_generated_prefixes"] == nil {
		t.Fatalf("unexpected scope item: %+v", scopeItem)
	}
	if gate.ApplyOpen || gate.RealAreaMatrixWriteOpened {
		t.Fatalf("apply must remain closed despite readiness: %+v", gate)
	}
}

func assertGeneratedApplyBetaItem(t *testing.T, gate GeneratedWriteApplyBetaGate, key string, status string) {
	t.Helper()
	item := generatedApplyBetaItemByKey(gate.Items, key)
	if item.Key == "" {
		t.Fatalf("item %s not found: %+v", key, gate.Items)
	}
	if item.Status != status {
		t.Fatalf("item %s status = %q, want %q: %+v", key, item.Status, status, item)
	}
}

func generatedApplyBetaItemByKey(items []GeneratedWriteApplyBetaGateItem, key string) GeneratedWriteApplyBetaGateItem {
	for _, item := range items {
		if item.Key == key {
			return item
		}
	}
	return GeneratedWriteApplyBetaGateItem{}
}
