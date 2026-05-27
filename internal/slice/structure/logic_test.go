package structure_test

import (
	"os"
	"testing"
	"time"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/structure"
)

// ── checkReadmePresent ────────────────────────────────────────────────────────

func TestCheckReadmePresent_happy(t *testing.T) {
	s := domain.RepoStructure{Files: []string{"README.md", "main.go"}}
	vs := structure.ExportCheckReadmePresent(s)
	if len(vs) != 0 {
		t.Fatalf("expected no violations, got %v", vs)
	}
}

func TestCheckReadmePresent_missing(t *testing.T) {
	s := domain.RepoStructure{Files: []string{"main.go"}}
	vs := structure.ExportCheckReadmePresent(s)
	if len(vs) == 0 {
		t.Fatal("expected violation, got none")
	}
	if vs[0].Code != "missing_readme" {
		t.Errorf("expected missing_readme, got %s", vs[0].Code)
	}
	if vs[0].Severity != "blocker" {
		t.Errorf("expected blocker, got %s", vs[0].Severity)
	}
}

// ── checkLinksResolve ─────────────────────────────────────────────────────────

func makeStructureWithDoc(docPath string, lines []string, files []string) domain.RepoStructure {
	doc := domain.MarkdownDoc{Path: docPath, Lines: lines}
	return domain.RepoStructure{
		Files: files,
		Docs:  []domain.MarkdownDoc{doc},
	}
}

func TestCheckLinksResolve_happy(t *testing.T) {
	s := makeStructureWithDoc(
		"README.md",
		[]string{"# Title", "[arch](docs/architecture.md)"},
		[]string{"README.md", "docs/architecture.md"},
	)
	vs := structure.ExportCheckLinksResolve(s)
	if len(vs) != 0 {
		t.Fatalf("expected no violations, got %v", vs)
	}
}

func TestCheckLinksResolve_brokenLink(t *testing.T) {
	s := makeStructureWithDoc(
		"README.md",
		[]string{"[guide](docs/guide.md)"},
		[]string{"README.md"},
	)
	vs := structure.ExportCheckLinksResolve(s)
	if len(vs) == 0 {
		t.Fatal("expected violation, got none")
	}
	if vs[0].Code != "broken_link" {
		t.Errorf("expected broken_link, got %s", vs[0].Code)
	}
	if vs[0].Severity != "blocker" {
		t.Errorf("expected blocker, got %s", vs[0].Severity)
	}
}

// ── checkDocDrift ─────────────────────────────────────────────────────────────

func makeCfg(days int) domain.Config {
	// NewConfig с пустым path даёт дефолт 90 дней; нам нужен кастомный —
	// используем хелпер, делегирующий к exported-конструктору.
	cfg, _ := domain.NewConfig(domain.Request{})
	_ = cfg
	return structure.ExportMakeConfig(days)
}

func TestCheckDocDrift_happy(t *testing.T) {
	recent := time.Now().Add(-10 * 24 * time.Hour)
	codeRecent := time.Now().Add(-5 * 24 * time.Hour)
	s := domain.RepoStructure{
		Files: []string{"README.md", "main.go"},
		Docs:  []domain.MarkdownDoc{{Path: "README.md"}},
		MTimes: map[string]time.Time{
			"README.md": recent,
			"main.go":   codeRecent,
		},
	}
	cfg := makeCfg(90)
	vs := structure.ExportCheckDocDrift(s, cfg)
	if len(vs) != 0 {
		t.Fatalf("expected no violations, got %v", vs)
	}
}

func TestCheckDocDrift_stale(t *testing.T) {
	old := time.Now().Add(-200 * 24 * time.Hour)
	codeRecent := time.Now().Add(-5 * 24 * time.Hour)
	s := domain.RepoStructure{
		Files: []string{"README.md", "main.go"},
		Docs:  []domain.MarkdownDoc{{Path: "README.md"}},
		MTimes: map[string]time.Time{
			"README.md": old,
			"main.go":   codeRecent,
		},
	}
	cfg := makeCfg(90)
	vs := structure.ExportCheckDocDrift(s, cfg)
	if len(vs) == 0 {
		t.Fatal("expected violation, got none")
	}
	if vs[0].Code != "doc_drift" {
		t.Errorf("expected doc_drift, got %s", vs[0].Code)
	}
	if vs[0].Severity != "warning" {
		t.Errorf("expected warning, got %s", vs[0].Severity)
	}
}

// ── buildReport ───────────────────────────────────────────────────────────────

func TestBuildReport_happy(t *testing.T) {
	s := domain.RepoStructure{Files: []string{"README.md"}}
	cfg := makeCfg(90)
	outcome := structure.ExportCheckStructure(s, cfg)

	// Нам нужен AuditTarget — создаём через реальную директорию.
	dir := t.TempDir()
	// Создаём README.md в tmp чтобы иметь корректный target.
	if err := os.WriteFile(dir+"/README.md", []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	req := domain.Request{Path: dir, Command: "structure"}
	target, err := domain.NewAuditTarget(req)
	if err != nil {
		t.Fatalf("NewAuditTarget: %v", err)
	}

	parts := domain.ReportParts{Layers: []domain.LayerOutcome{outcome}}
	report := structure.ExportBuildReport(parts, target, "structure")

	if report.Command != "structure" {
		t.Errorf("expected command=structure, got %s", report.Command)
	}
	if report.SchemaVersion != "1.0" {
		t.Errorf("expected schema_version=1.0, got %s", report.SchemaVersion)
	}
	if _, ok := report.Layers["L3"]; !ok {
		t.Error("expected L3 in layers")
	}
}

// ── checkStructure ────────────────────────────────────────────────────────────

func TestCheckStructure_happy(t *testing.T) {
	s := domain.RepoStructure{
		Files: []string{"README.md", "main.go"},
		Docs:  []domain.MarkdownDoc{{Path: "README.md", Lines: []string{"# Title"}}},
		MTimes: map[string]time.Time{
			"README.md": time.Now(),
			"main.go":   time.Now(),
		},
	}
	cfg := makeCfg(90)
	outcome := structure.ExportCheckStructure(s, cfg)
	if outcome.Result.Status != "pass" {
		t.Errorf("expected pass, got %s", outcome.Result.Status)
	}
}

func TestCheckStructure_blockerFail(t *testing.T) {
	s := domain.RepoStructure{
		Files: []string{"main.go"},
		Docs:  nil,
		MTimes: map[string]time.Time{
			"main.go": time.Now(),
		},
	}
	cfg := makeCfg(90)
	outcome := structure.ExportCheckStructure(s, cfg)
	if outcome.Result.Status != "fail" {
		t.Errorf("expected fail, got %s", outcome.Result.Status)
	}
}
