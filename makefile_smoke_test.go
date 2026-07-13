package areaflow_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestMakefileScriptReferencesExist(t *testing.T) {
	content, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	refs := uniqueMakefileScriptReferences(string(content))
	if len(refs) == 0 {
		t.Fatal("Makefile should reference at least one scripts/ path")
	}
	for _, ref := range refs {
		if _, err := os.Stat(ref); err != nil {
			t.Fatalf("Makefile references missing script %s: %v", ref, err)
		}
	}
}

func TestMakefileSmokeTargetsDryRun(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Fatalf("make must be available to validate smoke targets: %v", err)
	}
	content, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	targets := makefileSmokeTargets(string(content))
	if len(targets) == 0 {
		t.Fatal("Makefile should define at least one smoke target")
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			cmd := exec.Command("make", "-n", target)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("make -n %s failed: %v\n%s", target, err, output)
			}
		})
	}
}

func TestMakefileReferencedShellScriptsParse(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("bash must be available to validate shell scripts: %v", err)
	}
	content, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	for _, ref := range uniqueMakefileScriptReferences(string(content)) {
		if !strings.HasSuffix(ref, ".sh") {
			continue
		}
		t.Run(ref, func(t *testing.T) {
			cmd := exec.Command("bash", "-n", ref)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("bash -n %s failed: %v\n%s", ref, err, output)
			}
		})
	}
}

func TestDocumentedSmokeTargetsExist(t *testing.T) {
	content, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makefileTargets := makefileTargetSet(string(content))
	documentedTargets := documentedSmokeTargets(t)
	if len(documentedTargets) == 0 {
		t.Fatal("docs should reference at least one smoke target")
	}
	for _, target := range documentedTargets {
		if _, ok := makefileTargets[target]; !ok {
			t.Fatalf("documented smoke target %s is not defined in Makefile", target)
		}
	}
}

func TestSmokeScriptsHaveMakefileTargets(t *testing.T) {
	content, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makefileTargets := makefileTargetSet(string(content))
	for _, script := range smokeScripts(t) {
		target := strings.TrimSuffix(filepath.Base(script), ".sh")
		if _, ok := makefileTargets[target]; !ok {
			t.Fatalf("smoke script %s has no Makefile target %s", script, target)
		}
	}
}

func TestMakefileSmokeTargetsArePhony(t *testing.T) {
	content, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	phonyTargets := makefilePhonyTargetSet(string(content))
	for _, target := range makefileSmokeTargets(string(content)) {
		if _, ok := phonyTargets[target]; !ok {
			t.Fatalf("smoke target %s must be declared in .PHONY", target)
		}
	}
}

func uniqueMakefileScriptReferences(content string) []string {
	matches := regexp.MustCompile(`scripts/[A-Za-z0-9._/-]+`).FindAllString(content, -1)
	seen := map[string]struct{}{}
	for _, match := range matches {
		ref := strings.TrimRight(match, `"'`)
		if ref == "" {
			continue
		}
		seen[ref] = struct{}{}
	}
	refs := make([]string, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func makefileSmokeTargets(content string) []string {
	targets := []string{}
	for target := range makefileTargetSet(content) {
		if strings.HasPrefix(target, "smoke") {
			targets = append(targets, target)
		}
	}
	sort.Strings(targets)
	return targets
}

func makefileTargetSet(content string) map[string]struct{} {
	targetPattern := regexp.MustCompile(`(?m)^([A-Za-z0-9][A-Za-z0-9_-]*):(?:\s|$)`)
	seen := map[string]struct{}{}
	for _, match := range targetPattern.FindAllStringSubmatch(content, -1) {
		seen[match[1]] = struct{}{}
	}
	return seen
}

func makefilePhonyTargetSet(content string) map[string]struct{} {
	seen := map[string]struct{}{}
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, ".PHONY:") {
			continue
		}
		for _, target := range strings.Fields(strings.TrimPrefix(line, ".PHONY:")) {
			seen[target] = struct{}{}
		}
	}
	return seen
}

func documentedSmokeTargets(t *testing.T) []string {
	t.Helper()
	seen := map[string]struct{}{}
	for _, path := range documentationSearchPaths(t) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, target := range smokeTargetsFromText(string(content)) {
			seen[target] = struct{}{}
		}
	}
	targets := make([]string, 0, len(seen))
	for target := range seen {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

func smokeScripts(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob("scripts/smoke-*.sh")
	if err != nil {
		t.Fatalf("glob smoke scripts: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one smoke script")
	}
	sort.Strings(matches)
	return matches
}

func documentationSearchPaths(t *testing.T) []string {
	t.Helper()
	roots := []string{"README.md", "docs", "tasks", "workflow", "governance", "scripts"}
	paths := []string{}
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("stat %s: %v", root, err)
		}
		if !info.IsDir() {
			paths = append(paths, root)
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".sh") {
				paths = append(paths, path)
			}
			return nil
		}); err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	sort.Strings(paths)
	return paths
}

func smokeTargetsFromText(content string) []string {
	matches := regexp.MustCompile(`\bmake\s+(smoke[A-Za-z0-9_-]*)\b`).FindAllStringSubmatch(content, -1)
	seen := map[string]struct{}{}
	for _, match := range matches {
		seen[match[1]] = struct{}{}
	}
	targets := make([]string, 0, len(seen))
	for target := range seen {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}
