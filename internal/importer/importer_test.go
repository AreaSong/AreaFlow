package importer

import (
	"strings"
	"testing"

	"github.com/areasong/areaflow/internal/project"
)

func TestNormalizeOptions(t *testing.T) {
	options := normalizeOptions(Options{
		IdempotencyKey: " key-1 ",
		Actor:          " local-user ",
		Reason:         " import fixture ",
	})
	if options.IdempotencyKey != "key-1" || options.Actor != "local-user" || options.Reason != "import fixture" {
		t.Fatalf("unexpected normalized options: %+v", options)
	}

	defaults := normalizeOptions(Options{})
	if defaults.Actor != "local-user" || defaults.Reason != "read-only AreaMatrix metadata import" {
		t.Fatalf("unexpected default options: %+v", defaults)
	}
}

func TestImportRequestHashAndDefaultKey(t *testing.T) {
	record := project.Record{
		ID:       1,
		Key:      "areamatrix",
		Adapter:  "areamatrix",
		RootPath: "/tmp/areamatrix",
	}
	statusJSON := []byte(`{"version_count":2}`)
	options := normalizeOptions(Options{
		Actor:  "local-user",
		Reason: "fixture import",
	})
	first, err := importRequestHash(record, "source-a", statusJSON, options)
	if err != nil {
		t.Fatalf("first import hash failed: %v", err)
	}
	second, err := importRequestHash(record, "source-a", statusJSON, options)
	if err != nil {
		t.Fatalf("second import hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("same import request hash differed: %s != %s", first, second)
	}

	options.Reason = "different import"
	changed, err := importRequestHash(record, "source-a", statusJSON, options)
	if err != nil {
		t.Fatalf("changed import hash failed: %v", err)
	}
	if first == changed {
		t.Fatalf("import request hash should include audit reason")
	}

	firstKey := importIdempotencyKey(record, "source-a")
	secondKey := importIdempotencyKey(record, "source-a")
	if firstKey == secondKey {
		t.Fatalf("default import keys should be unique for repeated imports: %s", firstKey)
	}
	if !strings.HasPrefix(firstKey, "project.import:areamatrix:source-a:") {
		t.Fatalf("unexpected default import key: %s", firstKey)
	}
}
