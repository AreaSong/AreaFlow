package project

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	areamatrixadapter "github.com/areasong/areaflow/internal/adapter/areamatrix"
	workflowprofile "github.com/areasong/areaflow/internal/workflow"
)

var areaMatrixStageContract = []string{
	"intake",
	"source_docs",
	"templates",
	"version_init",
	"discussion",
	"middle_layer",
	"changes",
	"plans",
	"drafts",
	"queue",
	"promotion_preview",
	"approval",
	"execution",
	"run",
	"projection",
	"closeout",
}

var areaMatrixGateContract = []string{
	"import_coverage",
	"write_permission",
	"status_mirror",
	"hash_drift",
	"stage_coverage",
	"profile_binding_drift",
	"discussion_gate",
	"plan_doctor",
	"draft_doctor",
	"queue_doctor",
	"promotion_preview",
	"approval_gate",
	"live_mapping_gate",
	"runner_gate",
	"checkpoint_gate",
	"projection_gate",
	"closeout_gate",
}

var areaMatrixItemStateContract = []string{
	"draft",
	"ready",
	"blocked",
	"deferred",
	"promoted",
	"running",
	"done",
	"superseded",
}

type profileTransitionContract struct {
	From         string
	To           string
	RequiredGate string
}

var areaMatrixTransitionContract = []profileTransitionContract{
	{From: "intake", To: "source_docs"},
	{From: "source_docs", To: "templates"},
	{From: "templates", To: "version_init"},
	{From: "version_init", To: "discussion"},
	{From: "discussion", To: "middle_layer", RequiredGate: "discussion_gate"},
	{From: "middle_layer", To: "changes"},
	{From: "changes", To: "plans", RequiredGate: "plan_doctor"},
	{From: "plans", To: "drafts", RequiredGate: "plan_doctor"},
	{From: "drafts", To: "queue", RequiredGate: "draft_doctor"},
	{From: "queue", To: "promotion_preview", RequiredGate: "queue_doctor"},
	{From: "promotion_preview", To: "approval", RequiredGate: "promotion_preview"},
	{From: "approval", To: "execution", RequiredGate: "live_mapping_gate"},
	{From: "execution", To: "run", RequiredGate: "runner_gate"},
	{From: "run", To: "projection", RequiredGate: "checkpoint_gate"},
	{From: "projection", To: "closeout", RequiredGate: "projection_gate"},
}

var areaMatrixHardRuleContract = []string{
	"discussion_gate must pass before changes.",
	"plan_doctor must pass before drafts.",
	"draft_doctor must pass before queue.",
	"queue_doctor must pass before promotion_preview.",
	"promotion_preview pass is not apply.",
	"approval approved is not execution.",
	"live_mapping_gate pass is not cutover by itself.",
	"runner_gate must pass before real execution.",
	"projection_gate must pass before closeout.",
	"closeout_gate must prove evidence before done.",
}

var areaMatrixWriteRequiresContract = []string{
	"capability",
	"path_allowlist",
	"gate_result",
	"approval_record",
	"audit_event",
}

var pluginManifestDraftRequiredFields = []string{
	"package_id",
	"package_type",
	"display_name",
	"version",
	"publisher",
	"license",
	"source_uri",
	"package_hash",
	"signature",
	"compatibility.areaflow_min_version",
	"compatibility.areaflow_max_version",
	"compatibility.api_contract_version",
	"capabilities_requested",
	"resources_requested",
	"commands_requested",
	"network_access",
	"secret_refs_requested",
	"artifact_access",
	"project_write_access",
	"sandbox_policy",
	"install_steps",
	"disable_steps",
	"revoke_steps",
	"migration_steps",
	"rollback_steps",
	"conformance_checks",
	"audit_actions",
}

type pluginSeedPackage struct {
	PackageID          string
	PackageType        string
	DisplayName        string
	RegistryState      string
	MetadataOnly       bool
	InstallEnabled     bool
	EnableEnabled      bool
	ExecuteEnabled     bool
	RemoteFetchEnabled bool
	ProjectWriteAccess bool
}

type pluginMarketplaceBoundary struct {
	SeedCatalog                 []pluginSeedPackage
	AllowedRegistryStates       []string
	DeferredRegistryStates      []string
	ManifestDraftRequiredFields []string
	NoExecutionFacts            map[string]bool
	UnknownExecutionRung        string
}

type ConformanceOptions struct {
	GeneratedAt       time.Time
	ProfileRoot       string
	SnapshotLoadError string
}

type ConformanceCheck struct {
	Key      string
	Category string
	Status   string
	Message  string
	Metadata map[string]any
}

type ConformanceReport struct {
	Status      string
	Mode        string
	Project     Record
	ProfileID   string
	Adapter     string
	ProfileHash string
	StageCount  int
	GateCount   int
	Checks      []ConformanceCheck
	GeneratedAt time.Time
}

func (s Store) ConformanceCheck(ctx context.Context, record Record, options ConformanceOptions) (ConformanceReport, error) {
	options = normalizeConformanceOptions(options)
	root := strings.TrimSpace(options.ProfileRoot)
	if root == "" {
		var err error
		root, err = workflowProfileRoot()
		if err != nil {
			return ConformanceReport{}, err
		}
	}
	loaded, err := workflowprofile.LoadBuiltInProfile(root, record.WorkflowProfile)
	if err != nil {
		return ConformanceReport{}, err
	}
	var snapshot *areamatrixadapter.Snapshot
	if strings.TrimSpace(record.Adapter) == "areamatrix" && strings.TrimSpace(record.RootPath) != "" {
		loadedSnapshot, err := areamatrixadapter.Load(record.RootPath)
		if err != nil {
			options.SnapshotLoadError = err.Error()
		} else {
			snapshot = &loadedSnapshot
		}
	}
	config, hasConfig, err := s.ActiveProjectConfig(ctx, record.ID)
	if err != nil {
		return ConformanceReport{}, err
	}
	return BuildConformanceReport(record, loaded, snapshot, config, hasConfig, options), nil
}

func normalizeConformanceOptions(options ConformanceOptions) ConformanceOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildConformanceReport(record Record, loaded workflowprofile.LoadedProfile, snapshot *areamatrixadapter.Snapshot, config ProjectConfigRecord, hasConfig bool, options ConformanceOptions) ConformanceReport {
	options = normalizeConformanceOptions(options)
	profile := loaded.Profile
	report := ConformanceReport{
		Status:      "pass",
		Mode:        "read_only_adapter_profile_conformance",
		Project:     record,
		ProfileID:   profile.ProfileID,
		Adapter:     record.Adapter,
		ProfileHash: loaded.SHA256,
		StageCount:  len(profile.Stages),
		GateCount:   len(profile.Gates),
		GeneratedAt: options.GeneratedAt,
	}
	report.addCheck(checkProjectAdapterProfile(record, profile, loaded))
	report.addCheck(checkProfileLoad(loaded))
	report.addCheck(checkProfileValidate(profile, loaded.Warnings))
	report.addCheck(checkProfileItemStateContract(profile))
	report.addCheck(checkProfileStageContract(profile))
	report.addCheck(checkProfileGateContract(profile))
	report.addCheck(checkProfileTransitionContract(profile))
	report.addCheck(checkProfileHardRuleContract(profile))
	report.addCheck(checkProfilePermissionPolicyContract(profile))
	report.addCheck(checkProfileArtifactPolicyContract(profile))
	report.addCheck(checkProfileCutoverPolicyContract(profile))
	report.addCheck(checkAdapterSnapshot(record, snapshot, options.SnapshotLoadError))
	report.addCheck(checkAdapterProfileBoundary(record, profile))
	pluginBoundary := defaultPluginMarketplaceBoundary()
	report.addCheck(checkPluginSeedCatalogContract(pluginBoundary))
	report.addCheck(checkPluginManifestDraftContract(pluginBoundary))
	report.addCheck(checkPluginNoExecutionBoundary(pluginBoundary))
	report.addCheck(checkProjectConfigPolicy(record, config, hasConfig))
	return report
}

func defaultPluginMarketplaceBoundary() pluginMarketplaceBoundary {
	return pluginMarketplaceBoundary{
		SeedCatalog: []pluginSeedPackage{
			{
				PackageID:     "areamatrix-adapter",
				PackageType:   "adapter",
				DisplayName:   "AreaMatrix Adapter",
				RegistryState: "built_in",
				MetadataOnly:  true,
			},
			{
				PackageID:     "areamatrix-workflow-profile",
				PackageType:   "workflow_profile",
				DisplayName:   "AreaMatrix Workflow Profile",
				RegistryState: "built_in",
				MetadataOnly:  true,
			},
			{
				PackageID:     "areamatrix-template-catalog",
				PackageType:   "template",
				DisplayName:   "AreaMatrix Template Catalog",
				RegistryState: "seed",
				MetadataOnly:  true,
			},
		},
		AllowedRegistryStates:       []string{"built_in", "seed"},
		DeferredRegistryStates:      []string{"candidate", "verified", "enabled", "disabled", "suspended"},
		ManifestDraftRequiredFields: pluginManifestDraftRequiredFields,
		NoExecutionFacts: map[string]bool{
			"third_party_install_enabled": false,
			"third_party_enable_enabled":  false,
			"third_party_execute_enabled": false,
			"dynamic_loader_enabled":      false,
			"remote_fetch_enabled":        false,
			"capability_grant_enabled":    false,
			"secret_resolve_enabled":      false,
			"plugin_command_enabled":      false,
			"project_write_enabled":       false,
			"database_write_enabled":      false,
			"artifact_write_enabled":      false,
			"network_access_enabled":      false,
			"command_request_created":     false,
		},
		UnknownExecutionRung: "v1.x-rung-14",
	}
}

func (r *ConformanceReport) addCheck(check ConformanceCheck) {
	r.Checks = append(r.Checks, check)
	if worseConformanceStatus(check.Status, r.Status) {
		r.Status = check.Status
	}
}

func checkProjectAdapterProfile(record Record, profile workflowprofile.Profile, loaded workflowprofile.LoadedProfile) ConformanceCheck {
	failures := []string{}
	if strings.TrimSpace(record.Adapter) == "" {
		failures = append(failures, "project_adapter_missing")
	}
	if strings.TrimSpace(record.WorkflowProfile) == "" {
		failures = append(failures, "project_workflow_profile_missing")
	}
	if profile.AdapterDefaults.Adapter != "" && record.Adapter != profile.AdapterDefaults.Adapter {
		failures = append(failures, "adapter_mismatch")
	}
	if profile.AdapterDefaults.WorkflowProfile != "" && record.WorkflowProfile != profile.AdapterDefaults.WorkflowProfile {
		failures = append(failures, "workflow_profile_mismatch")
	}
	if profile.ProfileID != "" && record.WorkflowProfile != profile.ProfileID {
		failures = append(failures, "profile_id_mismatch")
	}
	metadata := map[string]any{
		"project_adapter":          record.Adapter,
		"project_workflow_profile": record.WorkflowProfile,
		"profile_id":               profile.ProfileID,
		"profile_version":          profile.ProfileVersion,
		"profile_path":             loaded.Path,
		"profile_hash":             loaded.SHA256,
		"adapter_defaults": map[string]any{
			"adapter":          profile.AdapterDefaults.Adapter,
			"workflow_profile": profile.AdapterDefaults.WorkflowProfile,
		},
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("project_adapter_profile", "binding", "fail", "project adapter/profile binding does not match loaded workflow profile defaults", metadata)
	}
	return conformanceCheck("project_adapter_profile", "binding", "pass", "project adapter/profile binding matches loaded workflow profile defaults", metadata)
}

func checkProfileLoad(loaded workflowprofile.LoadedProfile) ConformanceCheck {
	failures := []string{}
	if strings.TrimSpace(loaded.Path) == "" {
		failures = append(failures, "profile_path_missing")
	}
	if len(strings.TrimSpace(loaded.SHA256)) != 64 {
		failures = append(failures, "profile_hash_invalid")
	}
	metadata := map[string]any{
		"profile_path": loaded.Path,
		"profile_hash": loaded.SHA256,
		"profile_id":   loaded.Profile.ProfileID,
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("profile_load", "profile", "fail", "built-in workflow profile did not load with a stable sha256 hash", metadata)
	}
	return conformanceCheck("profile_load", "profile", "pass", "built-in workflow profile loaded with a stable sha256 hash", metadata)
}

func checkProfileValidate(profile workflowprofile.Profile, warnings []string) ConformanceCheck {
	metadata := map[string]any{
		"profile_id":      profile.ProfileID,
		"profile_version": profile.ProfileVersion,
		"warnings":        warnings,
	}
	if len(warnings) > 0 {
		return conformanceCheck("profile_validate", "profile", "warn", "workflow profile is valid but has warnings", metadata)
	}
	return conformanceCheck("profile_validate", "profile", "pass", "workflow profile validates without warnings", metadata)
}

func checkProfileItemStateContract(profile workflowprofile.Profile) ConformanceCheck {
	actual := make([]string, 0, len(profile.ItemStates))
	for _, state := range profile.ItemStates {
		actual = append(actual, strings.TrimSpace(state))
	}
	failures := []string{}
	if len(actual) != len(areaMatrixItemStateContract) {
		failures = append(failures, fmt.Sprintf("item_state_count:%d", len(actual)))
	}
	for index, expected := range areaMatrixItemStateContract {
		if index >= len(actual) || actual[index] != expected {
			failures = append(failures, fmt.Sprintf("item_state_order:%d:%s", index, expected))
		}
	}
	metadata := map[string]any{
		"expected_item_states": areaMatrixItemStateContract,
		"actual_item_states":   actual,
		"expected_count":       len(areaMatrixItemStateContract),
		"actual_count":         len(actual),
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("profile_item_state_contract", "profile", "fail", "AreaMatrix workflow profile item state contract drifted from the platform baseline", metadata)
	}
	return conformanceCheck("profile_item_state_contract", "profile", "pass", "AreaMatrix workflow profile exposes the expected item state contract", metadata)
}

func checkProfileStageContract(profile workflowprofile.Profile) ConformanceCheck {
	actual := make([]string, 0, len(profile.Stages))
	for _, stage := range profile.Stages {
		actual = append(actual, stage.Name)
	}
	status := "pass"
	message := "AreaMatrix workflow profile exposes the expected ordered stage contract"
	failures := []string{}
	if len(actual) != len(areaMatrixStageContract) {
		status = "fail"
		failures = append(failures, fmt.Sprintf("stage_count:%d", len(actual)))
	}
	for index, expected := range areaMatrixStageContract {
		if index >= len(actual) || actual[index] != expected {
			status = "fail"
			failures = append(failures, fmt.Sprintf("stage_order:%d:%s", index, expected))
		}
	}
	metadata := map[string]any{
		"expected_stages": areaMatrixStageContract,
		"actual_stages":   actual,
		"expected_count":  len(areaMatrixStageContract),
		"actual_count":    len(actual),
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		message = "AreaMatrix workflow profile stage contract does not match the platform expectation"
	}
	return conformanceCheck("profile_stage_contract", "profile", status, message, metadata)
}

func checkProfileGateContract(profile workflowprofile.Profile) ConformanceCheck {
	gates := map[string]bool{}
	actual := make([]string, 0, len(profile.Gates))
	for _, gate := range profile.Gates {
		gates[gate.Name] = true
		actual = append(actual, gate.Name)
	}
	missing := []string{}
	for _, expected := range areaMatrixGateContract {
		if !gates[expected] {
			missing = append(missing, expected)
		}
	}
	metadata := map[string]any{
		"expected_gates": areaMatrixGateContract,
		"actual_gates":   actual,
		"expected_count": len(areaMatrixGateContract),
		"actual_count":   len(actual),
		"missing_gates":  missing,
	}
	if len(missing) > 0 {
		return conformanceCheck("profile_gate_contract", "profile", "fail", "AreaMatrix workflow profile is missing required gate names", metadata)
	}
	return conformanceCheck("profile_gate_contract", "profile", "pass", "AreaMatrix workflow profile exposes the expected gate contract", metadata)
}

func checkProfileTransitionContract(profile workflowprofile.Profile) ConformanceCheck {
	actual := make([]profileTransitionContract, 0, len(profile.Transitions))
	for _, transition := range profile.Transitions {
		actual = append(actual, profileTransitionContract{
			From:         transition.From,
			To:           transition.To,
			RequiredGate: transition.RequiredGate,
		})
	}
	failures := []string{}
	if len(actual) != len(areaMatrixTransitionContract) {
		failures = append(failures, fmt.Sprintf("transition_count:%d", len(actual)))
	}
	for index, expected := range areaMatrixTransitionContract {
		if index >= len(actual) {
			failures = append(failures, fmt.Sprintf("transition_missing:%d:%s->%s", index, expected.From, expected.To))
			continue
		}
		if actual[index].From != expected.From || actual[index].To != expected.To {
			failures = append(failures, fmt.Sprintf("transition_order:%d:%s->%s", index, expected.From, expected.To))
		}
		if actual[index].RequiredGate != expected.RequiredGate {
			failures = append(failures, fmt.Sprintf("transition_gate:%d:%s", index, expected.RequiredGate))
		}
	}
	metadata := map[string]any{
		"expected_transitions": transitionSummaries(areaMatrixTransitionContract),
		"actual_transitions":   transitionSummaries(actual),
		"expected_count":       len(areaMatrixTransitionContract),
		"actual_count":         len(actual),
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("profile_transition_contract", "profile", "fail", "AreaMatrix workflow profile transition contract drifted from the platform baseline", metadata)
	}
	return conformanceCheck("profile_transition_contract", "profile", "pass", "AreaMatrix workflow profile exposes the expected ordered transition contract", metadata)
}

func checkProfileHardRuleContract(profile workflowprofile.Profile) ConformanceCheck {
	rules := map[string]bool{}
	actual := make([]string, 0, len(profile.HardRules))
	for _, rule := range profile.HardRules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		rules[rule] = true
		actual = append(actual, rule)
	}
	missing := []string{}
	for _, expected := range areaMatrixHardRuleContract {
		if !rules[expected] {
			missing = append(missing, expected)
		}
	}
	metadata := map[string]any{
		"expected_hard_rules": areaMatrixHardRuleContract,
		"actual_hard_rules":   actual,
		"expected_count":      len(areaMatrixHardRuleContract),
		"actual_count":        len(actual),
		"missing_hard_rules":  missing,
	}
	if len(missing) > 0 {
		metadata["failures"] = missing
		return conformanceCheck("profile_hard_rule_contract", "profile", "fail", "AreaMatrix workflow profile is missing required hard rules", metadata)
	}
	return conformanceCheck("profile_hard_rule_contract", "profile", "pass", "AreaMatrix workflow profile preserves the required hard rules", metadata)
}

func checkProfileArtifactPolicyContract(profile workflowprofile.Profile) ConformanceCheck {
	policy := profile.ArtifactPolicy
	failures := []string{}
	if policy.MetadataSource != "postgres" {
		failures = append(failures, "metadata_source_mismatch")
	}
	if policy.ContentSource != "artifact_store" {
		failures = append(failures, "content_source_mismatch")
	}
	if policy.SourceDocsOwner != "managed_project" {
		failures = append(failures, "source_docs_owner_mismatch")
	}
	if policy.GeneratedOutputOwner != "areaflow" {
		failures = append(failures, "generated_output_owner_mismatch")
	}
	if policy.DefaultContentBackend != "local" {
		failures = append(failures, "default_content_backend_mismatch")
	}
	metadata := map[string]any{
		"expected": map[string]any{
			"metadata_source":         "postgres",
			"content_source":          "artifact_store",
			"source_docs_owner":       "managed_project",
			"generated_output_owner":  "areaflow",
			"default_content_backend": "local",
		},
		"actual": map[string]any{
			"metadata_source":         policy.MetadataSource,
			"content_source":          policy.ContentSource,
			"source_docs_owner":       policy.SourceDocsOwner,
			"generated_output_owner":  policy.GeneratedOutputOwner,
			"default_content_backend": policy.DefaultContentBackend,
		},
		"metadata_in_postgres":        policy.MetadataSource == "postgres",
		"content_in_artifact_store":   policy.ContentSource == "artifact_store",
		"generated_owned_by_platform": policy.GeneratedOutputOwner == "areaflow",
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("profile_artifact_policy_contract", "profile", "fail", "AreaMatrix workflow profile artifact policy drifted from the platform baseline", metadata)
	}
	return conformanceCheck("profile_artifact_policy_contract", "profile", "pass", "AreaMatrix workflow profile keeps artifact ownership and storage policy stable", metadata)
}

func checkProfilePermissionPolicyContract(profile workflowprofile.Profile) ConformanceCheck {
	actual := make([]string, 0, len(profile.Permissions.WriteRequires))
	for _, guard := range profile.Permissions.WriteRequires {
		actual = append(actual, strings.TrimSpace(guard))
	}
	failures := []string{}
	if profile.Permissions.DefaultMode != "readonly" {
		failures = append(failures, "default_mode_not_readonly")
	}
	if len(actual) != len(areaMatrixWriteRequiresContract) {
		failures = append(failures, fmt.Sprintf("write_requires_count:%d", len(actual)))
	}
	for index, expected := range areaMatrixWriteRequiresContract {
		if index >= len(actual) || actual[index] != expected {
			failures = append(failures, fmt.Sprintf("write_requires_order:%d:%s", index, expected))
		}
	}
	metadata := map[string]any{
		"expected_default_mode":    "readonly",
		"actual_default_mode":      profile.Permissions.DefaultMode,
		"expected_write_requires":  areaMatrixWriteRequiresContract,
		"actual_write_requires":    actual,
		"expected_guard_count":     len(areaMatrixWriteRequiresContract),
		"actual_guard_count":       len(actual),
		"requires_capability":      containsStringValue(actual, "capability"),
		"requires_path_allowlist":  containsStringValue(actual, "path_allowlist"),
		"requires_gate_result":     containsStringValue(actual, "gate_result"),
		"requires_approval_record": containsStringValue(actual, "approval_record"),
		"requires_audit_event":     containsStringValue(actual, "audit_event"),
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("profile_permission_policy_contract", "profile", "fail", "AreaMatrix workflow profile permission policy drifted from the platform baseline", metadata)
	}
	return conformanceCheck("profile_permission_policy_contract", "profile", "pass", "AreaMatrix workflow profile preserves the required write guard order", metadata)
}

func checkProfileCutoverPolicyContract(profile workflowprofile.Profile) ConformanceCheck {
	policy := profile.Cutover
	failures := []string{}
	if policy.Strategy != "import_mirror_shadow_cutover_archive" {
		failures = append(failures, "strategy_mismatch")
	}
	if policy.V04Scope != "authoring_source_of_truth_only" {
		failures = append(failures, "v0_4_scope_mismatch")
	}
	if policy.ExecutionCutoverPhase != "v0.6" {
		failures = append(failures, "execution_cutover_phase_mismatch")
	}
	metadata := map[string]any{
		"expected": map[string]any{
			"strategy":                "import_mirror_shadow_cutover_archive",
			"v0_4_scope":              "authoring_source_of_truth_only",
			"execution_cutover_phase": "v0.6",
		},
		"actual": map[string]any{
			"strategy":                policy.Strategy,
			"v0_4_scope":              policy.V04Scope,
			"execution_cutover_phase": policy.ExecutionCutoverPhase,
		},
		"authoring_cutover_only": policy.V04Scope == "authoring_source_of_truth_only",
		"execution_not_v04":      policy.ExecutionCutoverPhase != "v0.4",
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("profile_cutover_policy_contract", "profile", "fail", "AreaMatrix workflow profile cutover policy drifted from the platform baseline", metadata)
	}
	return conformanceCheck("profile_cutover_policy_contract", "profile", "pass", "AreaMatrix workflow profile preserves authoring/execution cutover separation", metadata)
}

func checkAdapterSnapshot(record Record, snapshot *areamatrixadapter.Snapshot, loadError string) ConformanceCheck {
	metadata := map[string]any{
		"adapter":   record.Adapter,
		"root_path": record.RootPath,
		"read_only": true,
	}
	if strings.TrimSpace(record.Adapter) != "areamatrix" {
		metadata["reason"] = "adapter_specific_snapshot_not_required"
		return conformanceCheck("adapter_snapshot", "adapter", "skipped", "project adapter does not require the AreaMatrix snapshot contract", metadata)
	}
	if strings.TrimSpace(record.RootPath) == "" {
		metadata["failures"] = []string{"root_path_missing"}
		return conformanceCheck("adapter_snapshot", "adapter", "fail", "AreaMatrix adapter snapshot requires a project root path", metadata)
	}
	if strings.TrimSpace(loadError) != "" {
		metadata["failures"] = []string{"snapshot_load_failed"}
		metadata["load_error"] = loadError
		return conformanceCheck("adapter_snapshot", "adapter", "fail", "AreaMatrix adapter snapshot could not be loaded", metadata)
	}
	if snapshot == nil {
		metadata["failures"] = []string{"snapshot_missing"}
		return conformanceCheck("adapter_snapshot", "adapter", "fail", "AreaMatrix adapter snapshot was not loaded", metadata)
	}
	failures := []string{}
	if len(snapshot.Versions) == 0 {
		failures = append(failures, "versions_empty")
	}
	if len(snapshot.Residuals) == 0 {
		failures = append(failures, "residuals_empty")
	}
	if len(snapshot.Artifacts) == 0 {
		failures = append(failures, "artifacts_empty")
	}
	if strings.TrimSpace(snapshot.StatusSourceHash) == "" {
		failures = append(failures, "status_source_hash_missing")
	}
	metadata["versions"] = len(snapshot.Versions)
	metadata["residuals"] = len(snapshot.Residuals)
	metadata["artifacts"] = len(snapshot.Artifacts)
	metadata["status_source_hash"] = snapshot.StatusSourceHash
	metadata["v1_execution_total"] = snapshot.TaskSummary.V1ExecutionTotal
	metadata["v1_execution_done"] = snapshot.TaskSummary.V1ExecutionDone
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("adapter_snapshot", "adapter", "fail", "AreaMatrix adapter snapshot is missing required inventory", metadata)
	}
	return conformanceCheck("adapter_snapshot", "adapter", "pass", "AreaMatrix adapter can load a read-only project snapshot", metadata)
}

func checkAdapterProfileBoundary(record Record, profile workflowprofile.Profile) ConformanceCheck {
	failures := []string{}
	if strings.TrimSpace(profile.AdapterDefaults.Adapter) == "" {
		failures = append(failures, "profile_adapter_default_missing")
	}
	if strings.TrimSpace(profile.AdapterDefaults.WorkflowProfile) == "" {
		failures = append(failures, "profile_workflow_default_missing")
	}
	if strings.TrimSpace(profile.ArtifactPolicy.MetadataSource) == "" {
		failures = append(failures, "artifact_metadata_policy_missing")
	}
	if strings.TrimSpace(profile.ArtifactPolicy.ContentSource) == "" {
		failures = append(failures, "artifact_content_policy_missing")
	}
	if strings.TrimSpace(profile.Permissions.DefaultMode) == "" {
		failures = append(failures, "permission_default_mode_missing")
	}
	if !profile.VersionBinding.FreezeProfileHash {
		failures = append(failures, "profile_hash_freeze_disabled")
	}
	metadata := map[string]any{
		"adapter":                      record.Adapter,
		"profile_id":                   profile.ProfileID,
		"adapter_defines_stages":       false,
		"profile_reads_disk":           false,
		"profile_executes_commands":    false,
		"profile_resolves_secrets":     false,
		"freeze_profile_hash":          profile.VersionBinding.FreezeProfileHash,
		"artifact_metadata_source":     profile.ArtifactPolicy.MetadataSource,
		"artifact_content_source":      profile.ArtifactPolicy.ContentSource,
		"permission_default_mode":      profile.Permissions.DefaultMode,
		"write_requires":               profile.Permissions.WriteRequires,
		"boundary_is_metadata_only":    true,
		"conformance_writes_project":   false,
		"conformance_runs_commands":    false,
		"conformance_resolves_secret":  false,
		"conformance_writes_database":  false,
		"conformance_reads_profile":    true,
		"conformance_reads_project_fs": record.Adapter == "areamatrix",
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("adapter_profile_boundary", "boundary", "fail", "adapter/profile boundary metadata is incomplete", metadata)
	}
	return conformanceCheck("adapter_profile_boundary", "boundary", "pass", "adapter/profile/core boundary is explicit and read-only", metadata)
}

func checkPluginSeedCatalogContract(boundary pluginMarketplaceBoundary) ConformanceCheck {
	failures := []string{}
	summaries := make([]map[string]any, 0, len(boundary.SeedCatalog))
	if len(boundary.SeedCatalog) == 0 {
		failures = append(failures, "seed_catalog_empty")
	}
	for _, item := range boundary.SeedCatalog {
		packageID := strings.TrimSpace(item.PackageID)
		packageType := strings.TrimSpace(item.PackageType)
		registryState := strings.TrimSpace(item.RegistryState)
		if packageID == "" {
			failures = append(failures, "package_id_missing")
		}
		if !containsStringValue([]string{"adapter", "workflow_profile", "template"}, packageType) {
			failures = append(failures, "package_type_not_seed_allowed:"+packageID+":"+packageType)
		}
		if !containsStringValue(boundary.AllowedRegistryStates, registryState) {
			failures = append(failures, "seed_state_not_allowed:"+packageID+":"+registryState)
		}
		if !item.MetadataOnly {
			failures = append(failures, "seed_not_metadata_only:"+packageID)
		}
		if item.InstallEnabled {
			failures = append(failures, "seed_install_enabled:"+packageID)
		}
		if item.EnableEnabled {
			failures = append(failures, "seed_enable_enabled:"+packageID)
		}
		if item.ExecuteEnabled {
			failures = append(failures, "seed_execute_enabled:"+packageID)
		}
		if item.RemoteFetchEnabled {
			failures = append(failures, "seed_remote_fetch_enabled:"+packageID)
		}
		if item.ProjectWriteAccess {
			failures = append(failures, "seed_project_write_enabled:"+packageID)
		}
		summaries = append(summaries, map[string]any{
			"package_id":           packageID,
			"package_type":         packageType,
			"display_name":         item.DisplayName,
			"registry_state":       registryState,
			"metadata_only":        item.MetadataOnly,
			"install_enabled":      item.InstallEnabled,
			"enable_enabled":       item.EnableEnabled,
			"execute_enabled":      item.ExecuteEnabled,
			"remote_fetch_enabled": item.RemoteFetchEnabled,
			"project_write_access": item.ProjectWriteAccess,
		})
	}
	metadata := map[string]any{
		"seed_catalog":             summaries,
		"seed_count":               len(boundary.SeedCatalog),
		"allowed_registry_states":  boundary.AllowedRegistryStates,
		"deferred_registry_states": boundary.DeferredRegistryStates,
		"read_only":                true,
		"metadata_only":            true,
		"marketplace_fetches":      false,
		"plugin_install_open":      false,
		"plugin_execution_open":    false,
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("plugin_seed_catalog_contract", "plugin_marketplace", "fail", "plugin seed catalog boundary drifted from the v1.0 metadata-only scope", metadata)
	}
	return conformanceCheck("plugin_seed_catalog_contract", "plugin_marketplace", "pass", "plugin seed catalog is limited to built-in/seed metadata", metadata)
}

func checkPluginManifestDraftContract(boundary pluginMarketplaceBoundary) ConformanceCheck {
	actual := make([]string, 0, len(boundary.ManifestDraftRequiredFields))
	for _, field := range boundary.ManifestDraftRequiredFields {
		field = strings.TrimSpace(field)
		if field != "" {
			actual = append(actual, field)
		}
	}
	missing := []string{}
	for _, field := range pluginManifestDraftRequiredFields {
		if !containsStringValue(actual, field) {
			missing = append(missing, field)
		}
	}
	metadata := map[string]any{
		"required_fields":               pluginManifestDraftRequiredFields,
		"actual_fields":                 actual,
		"required_count":                len(pluginManifestDraftRequiredFields),
		"actual_count":                  len(actual),
		"missing_fields":                missing,
		"manifest_presence_is_approval": false,
		"manifest_lint_only":            true,
		"install_steps_execute":         false,
		"migration_steps_execute":       false,
	}
	if len(missing) > 0 {
		metadata["failures"] = missing
		return conformanceCheck("plugin_manifest_draft_contract", "plugin_marketplace", "fail", "plugin manifest draft is missing required governance fields", metadata)
	}
	return conformanceCheck("plugin_manifest_draft_contract", "plugin_marketplace", "pass", "plugin manifest draft declares the required future governance fields without opening execution", metadata)
}

func checkPluginNoExecutionBoundary(boundary pluginMarketplaceBoundary) ConformanceCheck {
	keys := make([]string, 0, len(boundary.NoExecutionFacts))
	for key := range boundary.NoExecutionFacts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	failures := []string{}
	for _, key := range keys {
		if boundary.NoExecutionFacts[key] {
			failures = append(failures, key)
		}
	}
	if strings.TrimSpace(boundary.UnknownExecutionRung) != "v1.x-rung-14" {
		failures = append(failures, "unknown_execution_rung_mismatch")
	}
	metadata := map[string]any{
		"no_execution_facts":          boundary.NoExecutionFacts,
		"unknown_execution_rung":      boundary.UnknownExecutionRung,
		"unknown_plugin_deferred":     strings.TrimSpace(boundary.UnknownExecutionRung) == "v1.x-rung-14",
		"permission_bypass_allowed":   false,
		"command_api_bypass_allowed":  false,
		"project_config_expansion":    false,
		"area_matrix_write_attempted": false,
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("plugin_no_execution_boundary", "plugin_marketplace", "fail", "plugin marketplace boundary opened execution or privileged access unexpectedly", metadata)
	}
	return conformanceCheck("plugin_no_execution_boundary", "plugin_marketplace", "pass", "unknown plugin install/enable/execute remains deferred and cannot bypass permissions", metadata)
}

func checkProjectConfigPolicy(record Record, config ProjectConfigRecord, hasConfig bool) ConformanceCheck {
	metadata := map[string]any{
		"adapter":                      record.Adapter,
		"has_config":                   hasConfig,
		"config_path":                  config.ConfigPath,
		"config_hash":                  config.ConfigHash,
		"protocol_version":             config.ProtocolVersion,
		"read_only":                    true,
		"conformance_writes_project":   false,
		"conformance_runs_commands":    false,
		"conformance_resolves_secret":  false,
		"conformance_writes_database":  false,
		"conformance_reads_config":     hasConfig,
		"conformance_opens_execution":  false,
		"conformance_opens_task_loop":  false,
		"conformance_opens_engine":     false,
		"conformance_opens_secret":     false,
		"conformance_opens_network":    false,
		"conformance_opens_git":        false,
		"conformance_opens_project_fs": false,
	}
	if strings.TrimSpace(record.Adapter) != "areamatrix" {
		metadata["reason"] = "adapter_specific_config_policy_not_required"
		return conformanceCheck("project_config_policy", "config", "skipped", "project adapter does not require the AreaMatrix config policy contract", metadata)
	}
	if !hasConfig {
		metadata["failures"] = []string{"active_project_config_missing"}
		return conformanceCheck("project_config_policy", "config", "fail", "AreaMatrix conformance requires an active areaflow.yaml config snapshot", metadata)
	}

	capabilities := mapFromAny(config.Permissions["capabilities"])
	writePaths := conformanceStringSliceFromAny(config.Permissions["write_paths"])
	forbiddenPaths := conformanceStringSliceFromAny(config.Permissions["forbidden_paths"])
	commands := mapFromAny(config.Metadata["commands"])
	allowedCommands := conformanceStringSliceFromAny(commands["allowed"])
	forbiddenCommands := conformanceStringSliceFromAny(commands["forbidden"])
	schedulingCapabilities := conformanceStringSliceFromAny(config.Scheduling["required_capabilities"])
	engines := mapFromAny(config.Engines)
	engineProfiles := conformanceMapSliceFromAny(engines["profiles"])
	statusSummary := mapFromAny(config.StatusExport["human_summary"])
	projectMetadata := mapFromAny(config.Metadata["project"])

	failures := []string{}
	if config.ProtocolVersion != 1 {
		failures = append(failures, "protocol_version_not_v1")
	}
	if conformanceString(projectMetadata, "adapter") != "" && conformanceString(projectMetadata, "adapter") != record.Adapter {
		failures = append(failures, "metadata_project_adapter_mismatch")
	}
	if conformanceString(projectMetadata, "workflow_profile") != "" && conformanceString(projectMetadata, "workflow_profile") != record.WorkflowProfile {
		failures = append(failures, "metadata_project_workflow_profile_mismatch")
	}
	if conformanceString(config.Ownership, "mode") == "" {
		failures = append(failures, "ownership_mode_missing")
	}
	if conformanceString(config.Migration, "strategy") != "import_mirror_shadow_cutover_archive" {
		failures = append(failures, "migration_strategy_mismatch")
	}
	if conformanceString(config.Migration, "phase") == "" {
		failures = append(failures, "migration_phase_missing")
	}
	if !boolFromAny(capabilities["read_project"]) {
		failures = append(failures, "read_project_not_enabled")
	}
	if !boolFromAny(capabilities["write_status"]) {
		failures = append(failures, "write_status_not_enabled")
	}
	for _, capability := range []string{"write_workflow", "write_generated", "write_code", "run_commands", "manage_workers", "manage_git", "network", "use_secrets", "execute_agents"} {
		if boolFromAny(capabilities[capability]) {
			failures = append(failures, capability+"_unexpectedly_enabled")
		}
	}
	if !containsStringValue(writePaths, ".areaflow/status.json") {
		failures = append(failures, "status_projection_write_path_missing")
	}
	for _, path := range []string{"workflow/versions/*/execution/**", "workflow/versions/*/execution/_shared/progress.json", ".areamatrix/**", "**/*.sqlite", "**/*.db"} {
		if !containsStringValue(forbiddenPaths, path) {
			failures = append(failures, "forbidden_path_missing:"+path)
		}
	}
	if containsStringValue(allowedCommands, "./task-loop run") {
		failures = append(failures, "task_loop_run_allowed")
	}
	for _, command := range []string{"./task-loop run", "git reset --hard", "git checkout --", "rm -rf"} {
		if !containsStringValue(forbiddenCommands, command) {
			failures = append(failures, "forbidden_command_missing:"+command)
		}
	}
	if !containsStringValue(schedulingCapabilities, "read_project") || !containsStringValue(schedulingCapabilities, "write_artifacts") {
		failures = append(failures, "scheduling_required_capabilities_mismatch")
	}
	if conformanceInt(config.Scheduling, "max_parallel_tasks") != 1 {
		failures = append(failures, "max_parallel_tasks_not_one")
	}
	if conformanceString(config.Scheduling, "engine_profile") == "" {
		failures = append(failures, "engine_profile_missing")
	}
	for _, profile := range engineProfiles {
		if boolFromAny(profile["enabled"]) {
			failures = append(failures, "engine_profile_enabled:"+conformanceString(profile, "id"))
		}
	}
	if !boolFromAny(config.StatusExport["enabled"]) {
		failures = append(failures, "status_export_disabled")
	}
	if conformanceString(config.StatusExport, "path") != ".areaflow/status.json" {
		failures = append(failures, "status_export_path_mismatch")
	}
	if boolFromAny(statusSummary["enabled"]) {
		failures = append(failures, "workflow_readme_human_summary_enabled_without_gate")
	}

	metadata["ownership_mode"] = conformanceString(config.Ownership, "mode")
	metadata["migration_strategy"] = conformanceString(config.Migration, "strategy")
	metadata["migration_phase"] = conformanceString(config.Migration, "phase")
	metadata["enabled_capabilities"] = enabledCapabilityNames(capabilities)
	metadata["write_paths"] = writePaths
	metadata["forbidden_paths"] = forbiddenPaths
	metadata["allowed_commands"] = allowedCommands
	metadata["forbidden_commands"] = forbiddenCommands
	metadata["scheduling_required_capabilities"] = schedulingCapabilities
	metadata["max_parallel_tasks"] = conformanceInt(config.Scheduling, "max_parallel_tasks")
	metadata["engine_profiles"] = engineProfileSummaries(engineProfiles)
	metadata["status_export_path"] = conformanceString(config.StatusExport, "path")
	metadata["human_summary_enabled"] = boolFromAny(statusSummary["enabled"])

	if len(failures) > 0 {
		metadata["failures"] = failures
		return conformanceCheck("project_config_policy", "config", "fail", "areaflow.yaml policy does not match the current AreaMatrix safety baseline", metadata)
	}
	return conformanceCheck("project_config_policy", "config", "pass", "areaflow.yaml policy keeps AreaMatrix integration in the current safe baseline", metadata)
}

func conformanceCheck(key string, category string, status string, message string, metadata map[string]any) ConformanceCheck {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return ConformanceCheck{
		Key:      key,
		Category: category,
		Status:   status,
		Message:  message,
		Metadata: metadata,
	}
}

func worseConformanceStatus(candidate string, current string) bool {
	rank := map[string]int{"pass": 0, "skipped": 1, "warn": 2, "fail": 3}
	return rank[candidate] > rank[current]
}

func conformanceString(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func conformanceInt(values map[string]any, key string) int {
	value, ok := values[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func conformanceStringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func conformanceMapSliceFromAny(value any) []map[string]any {
	switch items := value.(type) {
	case []map[string]any:
		out := make([]map[string]any, 0, len(items))
		out = append(out, items...)
		return out
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, mapFromAny(item))
		}
		return out
	default:
		return nil
	}
}

func containsStringValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func enabledCapabilityNames(capabilities map[string]any) []string {
	names := make([]string, 0, len(capabilities))
	for name, enabled := range capabilities {
		if boolFromAny(enabled) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func engineProfileSummaries(profiles []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, map[string]any{
			"id":         conformanceString(profile, "id"),
			"provider":   conformanceString(profile, "provider"),
			"secret_ref": conformanceString(profile, "secret_ref"),
			"enabled":    boolFromAny(profile["enabled"]),
		})
	}
	return out
}

func transitionSummaries(transitions []profileTransitionContract) []map[string]any {
	out := make([]map[string]any, 0, len(transitions))
	for _, transition := range transitions {
		out = append(out, map[string]any{
			"from":          transition.From,
			"to":            transition.To,
			"required_gate": transition.RequiredGate,
		})
	}
	return out
}
