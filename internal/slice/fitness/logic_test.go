package fitness

import (
	"strings"
	"testing"
	"time"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── buildJTBDPromptSet ───────────────────────────────────────────────────────

func TestBuildJTBDPromptSet_happy(t *testing.T) {
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	docs := []domain.MarkdownDoc{
		{Path: "README.md", Lines: []string{"# Hello", "World"}},
	}
	set := buildJTBDPromptSet(docs, cfg)
	if len(set.Prompts()) != 4 {
		t.Fatalf("expected 4 prompts, got %d", len(set.Prompts()))
	}
	for _, p := range set.Prompts() {
		if p.Consumer() == "" {
			t.Error("consumer must not be empty")
		}
		if p.Text() == "" {
			t.Error("text must not be empty")
		}
		if p.Budget() <= 0 {
			t.Error("budget must be positive")
		}
	}
}

func TestBuildJTBDPromptSet_emptyDocs(t *testing.T) {
	cfg, err := domain.NewConfig(domain.Request{})
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	set := buildJTBDPromptSet(nil, cfg)
	if len(set.Prompts()) != 4 {
		t.Fatalf("expected 4 prompts even for empty docs, got %d", len(set.Prompts()))
	}
}

// ── scoreFitness ─────────────────────────────────────────────────────────────

func TestScoreFitness_happy(t *testing.T) {
	verdicts := []domain.LLMVerdict{
		{Consumer: "maintainer", RawStatus: "PASS", RawScore: 92, RawGaps: nil},
		{Consumer: "consumer", RawStatus: "PARTIAL", RawScore: 55, RawGaps: []string{"missing"}},
	}
	results := scoreFitness(verdicts)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "PASS" || results[0].Score != 92 {
		t.Errorf("result[0] = %+v", results[0])
	}
	if results[1].Status != "PARTIAL" || results[1].Score != 55 {
		t.Errorf("result[1] = %+v", results[1])
	}
}

func TestScoreFitness_invalidStatusBecomesPartial(t *testing.T) {
	verdicts := []domain.LLMVerdict{
		{Consumer: "agent", RawStatus: "UNKNOWN_STATUS", RawScore: 30},
	}
	results := scoreFitness(verdicts)
	if results[0].Status != "PARTIAL" {
		t.Errorf("invalid status should become PARTIAL, got %q", results[0].Status)
	}
}

// ── estimateTokens / promptSetTokens (формула payload-бюджета) ───────────────

func TestEstimateTokens_bytesOverFour(t *testing.T) {
	if got := estimateTokens(strings.Repeat("x", 400)); got != 100 {
		t.Errorf("estimateTokens(400 bytes) = %d, want 100", got)
	}
	if got := estimateTokens(""); got != 0 {
		t.Errorf("estimateTokens(\"\") = %d, want 0", got)
	}
}

func TestPromptSetTokens_sumsPrompts(t *testing.T) {
	set := domain.NewJTBDPromptSet([]domain.JTBDPrompt{
		domain.NewJTBDPrompt("a", strings.Repeat("x", 400), 4096),
		domain.NewJTBDPrompt("b", strings.Repeat("y", 800), 4096),
	})
	if got := promptSetTokens(set); got != 300 {
		t.Errorf("promptSetTokens = %d, want 300 (100+200)", got)
	}
}

// ── overTokenBudget (защитный лимит) ─────────────────────────────────────────

func TestOverTokenBudget(t *testing.T) {
	cases := []struct {
		total, limit int
		want         bool
	}{
		{100, 50, true},
		{50, 50, false},       // равно — не превышение
		{10, 50, false},       // под лимитом
		{1_000_000, 0, false}, // limit<=0 отключает проверку
	}
	for _, c := range cases {
		if got := overTokenBudget(c.total, c.limit); got != c.want {
			t.Errorf("overTokenBudget(%d,%d) = %v, want %v", c.total, c.limit, got, c.want)
		}
	}
}

// ── retryWait (бэкофф по Retry-After) ────────────────────────────────────────

func TestRetryWait_honorsRetryAfterHeader(t *testing.T) {
	got := retryWait("2", 5, time.Second, 30*time.Second)
	if got != 2*time.Second {
		t.Errorf("retryWait(Retry-After=2) = %v, want 2s (заголовок приоритетен)", got)
	}
}

func TestRetryWait_exponentialWhenNoHeader(t *testing.T) {
	base, cap := time.Second, 30*time.Second
	if got := retryWait("", 0, base, cap); got != base {
		t.Errorf("attempt 0 = %v, want %v", got, base)
	}
	if got := retryWait("", 2, base, cap); got != 4*time.Second {
		t.Errorf("attempt 2 = %v, want 4s (base*2^2)", got)
	}
}

func TestRetryWait_cappedBoth(t *testing.T) {
	cap := 5 * time.Second
	if got := retryWait("3600", 0, time.Second, cap); got != cap {
		t.Errorf("огромный Retry-After должен ограничиться cap, got %v", got)
	}
	if got := retryWait("", 20, time.Second, cap); got != cap {
		t.Errorf("экспонента должна ограничиться cap, got %v", got)
	}
}
