package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const sourceAlignmentBindingContractVersion = "E1_design_source_alignment:v1"

var sourceAlignmentProofStaticSourcePaths = []string{
	"docs/product/master-plan.md",
	"docs/product/platform-blueprint.md",
	"docs/product/phase-backlog.md",
	"docs/product/roadmap.md",
	"docs/milestones/README.md",
	"tasks/backlog/0-100-platform-backlog.md",
	"docs/development/task-backlog-status-audit.md",
	"docs/development/implementation-gap-audit.md",
}

var sourceAlignmentProofSourceGlobs = []string{
	"docs/architecture/*.md",
	"docs/migration/*.md",
}

var sourceAlignmentBindingComparisonKeys = []string{
	"source_alignment_binding_contract",
	"source_alignment_source_set_hash",
	"source_alignment_source_file_count",
	"source_alignment_missing_source_count",
	"source_alignment_unreadable_source_count",
}

func SourceAlignmentCurrentBinding() (map[string]any, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get source alignment binding root: %w", err)
	}
	root, err := sourceAlignmentRepoRootFromCwd(cwd)
	if err != nil {
		return nil, err
	}
	return SourceAlignmentCurrentBindingForRoot(root)
}

func SourceAlignmentCurrentBindingForRoot(root string) (map[string]any, error) {
	paths, err := sourceAlignmentBindingSourcePaths(root)
	if err != nil {
		return nil, err
	}
	hashes := map[string]string{}
	for _, path := range paths {
		hash, err := sourceAlignmentFileSHA256(root, path)
		if err != nil {
			return nil, err
		}
		hashes[path] = hash
	}
	return sourceAlignmentProofBindingMetadata(paths, hashes, 0, 0, true, nil), nil
}

func sourceAlignmentRepoRootFromCwd(cwd string) (string, error) {
	dir := filepath.Clean(cwd)
	for {
		if sourceAlignmentLooksLikeAreaFlowRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("source alignment binding root not found from %s", cwd)
		}
		dir = parent
	}
}

func sourceAlignmentLooksLikeAreaFlowRoot(dir string) bool {
	goMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil || !strings.Contains(string(goMod), "module github.com/areasong/areaflow") {
		return false
	}
	for _, path := range []string{
		"docs/product/master-plan.md",
		"docs/architecture/completion-audit-contract.md",
		"tasks/backlog/0-100-platform-backlog.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(path))); err != nil {
			return false
		}
	}
	return true
}

func sourceAlignmentBindingSourcePaths(root string) ([]string, error) {
	paths := append([]string{}, sourceAlignmentProofStaticSourcePaths...)
	for _, pattern := range sourceAlignmentProofSourceGlobs {
		cleanPattern, err := sourceAlignmentCleanRelativePath(pattern)
		if err != nil {
			return nil, err
		}
		matches, err := filepath.Glob(filepath.Join(root, cleanPattern))
		if err != nil {
			return nil, fmt.Errorf("expand source alignment binding glob %s: %w", pattern, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("source alignment binding glob matched no files: %s", pattern)
		}
		for _, match := range matches {
			relative, err := filepath.Rel(root, match)
			if err != nil {
				return nil, fmt.Errorf("relativize source alignment source %s: %w", match, err)
			}
			paths = append(paths, filepath.ToSlash(relative))
		}
	}
	return normalizeStringList(paths), nil
}

func sourceAlignmentFileSHA256(root string, relativePath string) (string, error) {
	cleanPath, err := sourceAlignmentCleanRelativePath(relativePath)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.Join(root, cleanPath))
	if err != nil {
		return "", fmt.Errorf("read source alignment binding source %s: %w", relativePath, err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func sourceAlignmentCleanRelativePath(relativePath string) (string, error) {
	cleanPath := filepath.Clean(filepath.FromSlash(relativePath))
	if filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source alignment binding path escapes root: %s", relativePath)
	}
	return cleanPath, nil
}

func sourceAlignmentProofBindingMetadata(paths []string, hashes map[string]string, missingCount int64, unreadableCount int64, pass bool, blockers []string) map[string]any {
	status := "fail"
	if pass {
		status = "pass"
	}
	paths = normalizeStringList(paths)
	return map[string]any{
		"source_alignment_binding_status":           status,
		"source_alignment_binding_blockers":         uniqueStrings(blockers),
		"source_alignment_binding_contract":         sourceAlignmentBindingContractVersion,
		"source_alignment_source_paths":             paths,
		"source_alignment_source_hashes":            copyStringMap(hashes),
		"source_alignment_source_set_hash":          sourceAlignmentSourceSetHash(paths, hashes),
		"source_alignment_source_file_count":        int64(len(paths)),
		"source_alignment_missing_source_count":     missingCount,
		"source_alignment_unreadable_source_count":  unreadableCount,
		"source_alignment_source_globs":             append([]string{}, sourceAlignmentProofSourceGlobs...),
		"source_alignment_static_source_path_count": int64(len(sourceAlignmentProofStaticSourcePaths)),
	}
}

func sourceAlignmentSourceSetHash(paths []string, hashes map[string]string) string {
	paths = normalizeStringList(paths)
	orderedHashes := map[string]string{}
	for _, path := range paths {
		orderedHashes[path] = hashes[path]
	}
	payload, err := json.Marshal(map[string]any{
		"contract":      sourceAlignmentBindingContractVersion,
		"source_paths":  paths,
		"source_hashes": orderedHashes,
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func addSourceAlignmentProofBindingMetadata(metadata map[string]any, options RecordSourceAlignmentProofOptions) {
	binding := map[string]any{}
	for key, value := range options.SourceAlignmentBinding {
		binding[key] = value
	}
	if options.ProofStatus != "complete" && len(binding) == 0 {
		binding["source_alignment_binding_status"] = "not_required"
		binding["source_alignment_binding_blockers"] = []string{}
	} else if blockers := sourceAlignmentProofOptionsBindingBlockers(options); len(blockers) > 0 {
		binding["source_alignment_binding_status"] = "fail"
		binding["source_alignment_binding_blockers"] = blockers
	} else if options.ProofStatus == "complete" {
		binding["source_alignment_binding_status"] = "pass"
		binding["source_alignment_binding_blockers"] = []string{}
	}
	for key, value := range binding {
		metadata[key] = value
	}
}

func sourceAlignmentProofOptionsBindingBlockers(options RecordSourceAlignmentProofOptions) []string {
	if len(options.SourceAlignmentBinding) == 0 {
		return []string{"source_alignment_binding_missing"}
	}
	blockers := sourceAlignmentProofMetadataBindingBlockers(options.SourceAlignmentBinding)
	if len(blockers) > 0 {
		return blockers
	}
	currentBinding, err := SourceAlignmentCurrentBinding()
	if err != nil {
		return []string{"source_alignment_current_binding_query_failed"}
	}
	return sourceAlignmentProofCurrentBindingBlockers(options.SourceAlignmentBinding, currentBinding)
}

func sourceAlignmentProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "source_alignment_binding_status") != "pass" {
		blockers = append(blockers, "source_alignment_binding_status_not_pass")
	}
	if metadataString(metadata, "source_alignment_binding_contract") != sourceAlignmentBindingContractVersion {
		blockers = append(blockers, "source_alignment_binding_contract_missing_or_mismatch")
	}
	paths := metadataStringSlice(metadata, "source_alignment_source_paths")
	if len(paths) == 0 {
		blockers = append(blockers, "source_alignment_source_paths_missing")
	}
	hashes := sourceAlignmentMetadataStringMap(metadata, "source_alignment_source_hashes")
	if len(hashes) == 0 {
		blockers = append(blockers, "source_alignment_source_hashes_missing")
	}
	for _, path := range paths {
		if !looksLikeSHA256(hashes[path]) {
			blockers = append(blockers, "source_alignment_source_hash_missing_or_invalid")
			break
		}
	}
	if metadataInt64(metadata, "source_alignment_source_file_count") != int64(len(normalizeStringList(paths))) || metadataInt64(metadata, "source_alignment_source_file_count") == 0 {
		blockers = append(blockers, "source_alignment_source_file_count_missing_or_mismatch")
	}
	if metadataInt64(metadata, "source_alignment_missing_source_count") != 0 {
		blockers = append(blockers, "source_alignment_missing_source_count_nonzero")
	}
	if metadataInt64(metadata, "source_alignment_unreadable_source_count") != 0 {
		blockers = append(blockers, "source_alignment_unreadable_source_count_nonzero")
	}
	expectedHash := sourceAlignmentSourceSetHash(paths, hashes)
	if !looksLikeSHA256(metadataString(metadata, "source_alignment_source_set_hash")) ||
		metadataString(metadata, "source_alignment_source_set_hash") != expectedHash {
		blockers = append(blockers, "source_alignment_source_set_hash_missing_or_mismatch")
	}
	return uniqueStrings(blockers)
}

func sourceAlignmentProofCurrentBindingBlockers(proofMetadata map[string]any, currentBinding map[string]any) []string {
	blockers := sourceAlignmentProofMetadataBindingBlockers(proofMetadata)
	if len(blockers) > 0 {
		return blockers
	}
	currentBlockers := sourceAlignmentProofMetadataBindingBlockers(currentBinding)
	if len(currentBlockers) > 0 {
		for _, blocker := range currentBlockers {
			blockers = append(blockers, "current_"+blocker)
		}
		return uniqueStrings(blockers)
	}
	for _, key := range sourceAlignmentBindingComparisonKeys {
		if metadataString(proofMetadata, key) != metadataString(currentBinding, key) && !sourceAlignmentMetadataInt64Key(key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
		if sourceAlignmentMetadataInt64Key(key) && metadataInt64(proofMetadata, key) != metadataInt64(currentBinding, key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
	}
	if !sameNormalizedStrings(metadataStringSlice(proofMetadata, "source_alignment_source_paths"), metadataStringSlice(currentBinding, "source_alignment_source_paths")) {
		blockers = append(blockers, "source_alignment_source_paths_current_mismatch")
	}
	if metadataString(proofMetadata, "source_alignment_source_set_hash") != metadataString(currentBinding, "source_alignment_source_set_hash") {
		blockers = append(blockers, "source_alignment_source_hashes_current_mismatch")
	}
	return uniqueStrings(blockers)
}

func sourceAlignmentMetadataInt64Key(key string) bool {
	switch key {
	case "source_alignment_source_file_count", "source_alignment_missing_source_count", "source_alignment_unreadable_source_count":
		return true
	default:
		return false
	}
}

func sourceAlignmentMetadataStringMap(metadata map[string]any, key string) map[string]string {
	out := map[string]string{}
	value, ok := metadata[key]
	if !ok || value == nil {
		return out
	}
	switch typed := value.(type) {
	case map[string]string:
		for key, value := range typed {
			out[key] = value
		}
	case map[string]any:
		for key, value := range typed {
			if text, ok := value.(string); ok {
				out[key] = text
			}
		}
	}
	return out
}

func copyStringMap(values map[string]string) map[string]string {
	out := map[string]string{}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out[key] = values[key]
	}
	return out
}
