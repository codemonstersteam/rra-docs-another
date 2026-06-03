package drift_test

import (
	"strings"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
	"github.com/codemonstersteam/rra-docs-another/internal/slice/drift"
)

// ── extractClaims ─────────────────────────────────────────────────────────────

func TestExtractClaims_happy(t *testing.T) {
	doc := domain.MarkdownDoc{
		Path:  "README.md",
		Lines: []string{"Конфигурация — `config/settings.yaml`.", "Обычный текст."},
	}
	s := domain.RepoStructure{Docs: []domain.MarkdownDoc{doc}}
	claims := drift.ExportExtractClaims(s)
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
	if claims[0].Kind != "link" || claims[0].Text != "config/settings.yaml" {
		t.Errorf("unexpected claim: %+v", claims[0])
	}
	if claims[0].File != "README.md" || claims[0].Line != 1 {
		t.Errorf("wrong location: file=%s line=%d", claims[0].File, claims[0].Line)
	}
}

func TestExtractClaims_noUtterances(t *testing.T) {
	doc := domain.MarkdownDoc{
		Path:  "README.md",
		Lines: []string{"Сервис `order-service` обрабатывает заказы.", "Без путей."},
	}
	s := domain.RepoStructure{Docs: []domain.MarkdownDoc{doc}}
	claims := drift.ExportExtractClaims(s)
	if len(claims) != 0 {
		t.Fatalf("expected 0 claims, got %d: %+v", len(claims), claims)
	}
}

// ── NewDriftCheck ─────────────────────────────────────────────────────────────

func TestNewDriftCheck_happy(t *testing.T) {
	s := domain.RepoStructure{Files: []string{"README.md"}}
	claims := []drift.Claim{{Kind: "link", Text: "docs/api.md", File: "README.md", Line: 1}}
	check := drift.NewDriftCheck(s, claims)
	// DriftCheck создан — проверяем через verifyClaims
	findings := drift.ExportVerifyClaims(check)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (docs/api.md missing), got %d", len(findings))
	}
}

// ── verifyClaims ──────────────────────────────────────────────────────────────

func TestVerifyClaims_happy(t *testing.T) {
	s := domain.RepoStructure{
		Files: []string{"README.md", "docs/api.md"},
	}
	claims := []drift.Claim{{Kind: "link", Text: "docs/api.md", File: "README.md", Line: 1}}
	check := drift.NewDriftCheck(s, claims)
	findings := drift.ExportVerifyClaims(check)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(findings), findings)
	}
}

func TestVerifyClaims_brokenPath(t *testing.T) {
	s := domain.RepoStructure{
		Files: []string{"README.md"},
	}
	claims := []drift.Claim{{Kind: "link", Text: "config/settings.yaml", File: "README.md", Line: 3}}
	check := drift.NewDriftCheck(s, claims)
	findings := drift.ExportVerifyClaims(check)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if !strings.Contains(findings[0].Reason, "config/settings.yaml") {
		t.Errorf("reason should mention path, got: %s", findings[0].Reason)
	}
}

func TestVerifyClaims_dependencyNotInManifest(t *testing.T) {
	s := domain.RepoStructure{
		Files:     []string{"README.md"},
		Manifests: map[string]string{"go.mod": "module example.com\n\ngo 1.23\n"},
	}
	claims := []drift.Claim{{Kind: "dependency", Text: "github.com/some/pkg", File: "README.md", Line: 5}}
	check := drift.NewDriftCheck(s, claims)
	findings := drift.ExportVerifyClaims(check)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if !strings.Contains(findings[0].Reason, "github.com/some/pkg") {
		t.Errorf("reason should mention pkg, got: %s", findings[0].Reason)
	}
}

// ── buildClaimPromptSet ───────────────────────────────────────────────────────

func TestBuildClaimPromptSet_happy(t *testing.T) {
	doc := domain.MarkdownDoc{
		Path:  "README.md",
		Lines: []string{"line1", "Конфигурация — `config/settings.yaml`.", "line3"},
	}
	s := domain.RepoStructure{Docs: []domain.MarkdownDoc{doc}}
	claims := []drift.Claim{{Kind: "link", Text: "config/settings.yaml", File: "README.md", Line: 2}}
	check := drift.NewDriftCheck(s, claims)
	cfg := drift.ExportDefaultCfg()
	set := drift.ExportBuildClaimPromptSet(check, cfg)
	if set.IsEmpty() {
		t.Fatal("expected non-empty ClaimPromptSet")
	}
	if len(set.Prompts()) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(set.Prompts()))
	}
}

func TestBuildClaimPromptSet_noEligible(t *testing.T) {
	s := domain.RepoStructure{}
	check := drift.NewDriftCheck(s, nil)
	cfg := drift.ExportDefaultCfg()
	set := drift.ExportBuildClaimPromptSet(check, cfg)
	if !set.IsEmpty() {
		t.Fatal("expected empty ClaimPromptSet")
	}
}

func TestBuildClaimPromptSet_capped(t *testing.T) {
	// Создаём 25 claims при MaxJudgeCalls=20 (дефолт).
	claims := make([]drift.Claim, 25)
	for i := range claims {
		claims[i] = drift.Claim{Kind: "link", Text: "a/b.md", File: "README.md", Line: i + 1}
	}
	s := domain.RepoStructure{
		Docs: []domain.MarkdownDoc{{Path: "README.md", Lines: make([]string, 30)}},
	}
	check := drift.NewDriftCheck(s, claims)
	cfg := drift.ExportDefaultCfg()
	set := drift.ExportBuildClaimPromptSet(check, cfg)
	if len(set.Prompts()) != 20 {
		t.Fatalf("expected 20 prompts (capped), got %d", len(set.Prompts()))
	}
}

// ── mergeSemanticFindings ─────────────────────────────────────────────────────

func TestMergeSemanticFindings_happy(t *testing.T) {
	verdicts := []domain.Verdict{{OK: true, Quote: ""}}
	findings := drift.ExportMergeSemanticFindings(verdicts)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestMergeSemanticFindings_failVerdict(t *testing.T) {
	verdicts := []domain.Verdict{{OK: false, Quote: "устаревшее описание"}}
	findings := drift.ExportMergeSemanticFindings(verdicts)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Reason != "устаревшее описание" {
		t.Errorf("expected quote as reason, got: %s", findings[0].Reason)
	}
}

func TestMergeSemanticFindings_empty(t *testing.T) {
	findings := drift.ExportMergeSemanticFindings(nil)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

// ── NewDriftReport ────────────────────────────────────────────────────────────

func TestNewDriftReport_happy(t *testing.T) {
	l6a := []drift.DriftFinding{{Reason: "path not found"}}
	report := drift.NewDriftReport(l6a, nil)
	// Проверяем через buildDriftOutcome: должен быть fail
	outcome := drift.ExportBuildDriftOutcome(report)
	if outcome.Result.Status != "fail" {
		t.Errorf("expected fail, got %s", outcome.Result.Status)
	}
}

// ── buildDriftOutcome ─────────────────────────────────────────────────────────

func TestBuildDriftOutcome_happy(t *testing.T) {
	report := drift.NewDriftReport(nil, nil)
	outcome := drift.ExportBuildDriftOutcome(report)
	if outcome.Result.Status != "pass" {
		t.Errorf("expected pass, got %s", outcome.Result.Status)
	}
	if outcome.Result.Name != "drift" {
		t.Errorf("expected name=drift, got %s", outcome.Result.Name)
	}
	if len(outcome.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(outcome.Violations))
	}
}

func TestBuildDriftOutcome_blockerFail(t *testing.T) {
	finding := drift.DriftFinding{
		Claim:  drift.Claim{Kind: "link", Text: "missing.md", File: "README.md", Line: 2},
		Reason: "путь не найден: missing.md",
	}
	report := drift.NewDriftReport([]drift.DriftFinding{finding}, nil)
	outcome := drift.ExportBuildDriftOutcome(report)
	if outcome.Result.Status != "fail" {
		t.Errorf("expected fail, got %s", outcome.Result.Status)
	}
	if len(outcome.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(outcome.Violations))
	}
	if outcome.Violations[0].Severity != "blocker" {
		t.Errorf("expected blocker, got %s", outcome.Violations[0].Severity)
	}
}

// ── isFilePath ────────────────────────────────────────────────────────────────

func TestIsFilePath_happy(t *testing.T) {
	valid := []string{
		"docs/adr",
		"docs/architecture.md",
		"internal/io/repostore.go",
		"config/settings.yaml",
		"component-tests/README.md",
		"cmd/api/main.go",
	}
	for _, s := range valid {
		if !drift.ExportIsFilePath(s) {
			t.Errorf("isFilePath(%q) = false, хотя должен быть валидным путём", s)
		}
	}
}

func TestIsFilePath_rejectGitRemote(t *testing.T) {
	cases := []string{
		"git@github.com:ubik-life/passkey-demo-api.git",
		"user@example.com/repo",
	}
	for _, s := range cases {
		if drift.ExportIsFilePath(s) {
			t.Errorf("isFilePath(%q) = true, git-remote/e-mail не должен быть путём", s)
		}
	}
}

func TestIsFilePath_rejectAbsolutePath(t *testing.T) {
	cases := []string{
		"/migrations",
		"/v1/users",
		"/contract-tests",
	}
	for _, s := range cases {
		if drift.ExportIsFilePath(s) {
			t.Errorf("isFilePath(%q) = true, абсолютный путь не должен матчиться", s)
		}
	}
}

func TestIsFilePath_rejectPlaceholder(t *testing.T) {
	cases := []string{
		"/v1/...",
		"a/.../b",
		"docs/.../readme",
	}
	for _, s := range cases {
		if drift.ExportIsFilePath(s) {
			t.Errorf("isFilePath(%q) = true, плейсхолдер ... не должен матчиться", s)
		}
	}
}
