package drift

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// extractClaims извлекает проверяемые утверждения из доков репозитория.
// link: относительные пути в inline-backticks (содержат "/", без glob/схем).
// dependency: имена модулей из go.mod, упомянутые в доках.
// Пути внутри fenced-блоков (```) не обрабатываются — высокий шанс ложных срабатываний.
func extractClaims(structure domain.RepoStructure) []Claim {
	pkgNames := goModPackages(structure.Manifests)

	var claims []Claim
	for _, doc := range structure.Docs {
		inFenced := false
		for i, line := range doc.Lines {
			lineNum := i + 1
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFenced = !inFenced
				continue
			}
			if inFenced {
				continue
			}
			for _, text := range inlineBacktickPaths(line) {
				claims = append(claims, Claim{
					Kind: "link",
					Text: text,
					File: doc.Path,
					Line: lineNum,
				})
			}
			for _, pkg := range pkgNames {
				if strings.Contains(line, pkg) {
					claims = append(claims, Claim{
						Kind: "dependency",
						Text: pkg,
						File: doc.Path,
						Line: lineNum,
					})
				}
			}
		}
	}
	return claims
}

// inlineBacktickPaths возвращает содержимое inline-backtick-кода, похожее на пути.
func inlineBacktickPaths(line string) []string {
	var result []string
	for {
		a := strings.Index(line, "`")
		if a < 0 {
			break
		}
		b := strings.Index(line[a+1:], "`")
		if b < 0 {
			break
		}
		content := line[a+1 : a+1+b]
		line = line[a+1+b+1:]
		if isFilePath(content) {
			result = append(result, content)
		}
	}
	return result
}

// isFilePath — истинно для строк, похожих на относительный путь к файлу:
// содержат "/", без "://", без glob-символов, пробелов и angle-bracket-шаблонов.
func isFilePath(s string) bool {
	return s != "" &&
		strings.Contains(s, "/") &&
		!strings.Contains(s, "://") &&
		!strings.HasPrefix(s, "-") &&
		!strings.ContainsAny(s, "* ?<>")
}

// goModPackages извлекает имена модулей из go.mod-манифеста в структуре репо.
func goModPackages(manifests map[string]string) []string {
	content, ok := manifests["go.mod"]
	if !ok {
		return nil
	}
	var pkgs []string
	inRequire := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "require (":
			inRequire = true
		case trimmed == ")":
			inRequire = false
		case strings.HasPrefix(trimmed, "require "):
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				pkgs = append(pkgs, parts[1])
			}
		case inRequire:
			parts := strings.Fields(trimmed)
			if len(parts) >= 1 && strings.Contains(parts[0], ".") && !strings.HasPrefix(parts[0], "//") {
				pkgs = append(pkgs, parts[0])
			}
		}
	}
	return pkgs
}

// verifyClaims механически проверяет утверждения (L6a, без ИИ).
// link: путь должен существовать в structure.Files (резолв от директории doc-файла).
// dependency: модуль должен упоминаться в structure.Manifests.
// Нарушение — только при механическом подтверждении.
func verifyClaims(check DriftCheck) []DriftFinding {
	fileSet := make(map[string]struct{}, len(check.structure.Files))
	for _, f := range check.structure.Files {
		fileSet[filepath.ToSlash(f)] = struct{}{}
	}

	var findings []DriftFinding
	for _, claim := range check.claims {
		switch claim.Kind {
		case "link":
			// Пробуем два варианта резолвинга: от корня репо (наиболее частое
			// соглашение в документации) и относительно директории doc-файла.
			rootResolved := filepath.ToSlash(filepath.Clean(claim.Text))
			docResolved := resolvePath(filepath.Dir(claim.File), claim.Text)
			if !pathOrDirExists(rootResolved, fileSet) && !pathOrDirExists(docResolved, fileSet) {
				findings = append(findings, DriftFinding{
					Claim:  claim,
					Reason: fmt.Sprintf("путь не найден: %s", rootResolved),
				})
			}
		case "dependency":
			if !dependencyInManifests(claim.Text, check.structure.Manifests) {
				findings = append(findings, DriftFinding{
					Claim:  claim,
					Reason: fmt.Sprintf("зависимость не найдена в манифестах: %s", claim.Text),
				})
			}
		}
	}
	return findings
}

// pathOrDirExists — истинно если resolved есть среди файлов или является
// директорией (т.е. существует хотя бы один файл с этим префиксом).
func pathOrDirExists(resolved string, fileSet map[string]struct{}) bool {
	if _, ok := fileSet[resolved]; ok {
		return true
	}
	prefix := resolved + "/"
	for f := range fileSet {
		if strings.HasPrefix(f, prefix) {
			return true
		}
	}
	return false
}

// resolvePath резолвит путь claim относительно директории doc-файла.
func resolvePath(docDir, claimPath string) string {
	if filepath.IsAbs(claimPath) {
		return filepath.ToSlash(claimPath[1:])
	}
	return filepath.ToSlash(filepath.Join(docDir, claimPath))
}

// dependencyInManifests проверяет, упоминается ли зависимость в манифестах.
func dependencyInManifests(dep string, manifests map[string]string) bool {
	for _, content := range manifests {
		if strings.Contains(content, dep) {
			return true
		}
	}
	return false
}

// buildClaimPromptSet формирует набор пар для семантического судьи (L6c).
// Все claim-kinds v1 считаются semantic-eligible.
// Cap: обрезает до cfg.MaxJudgeCalls с предупреждением (no silent caps).
func buildClaimPromptSet(check DriftCheck, cfg domain.Config) domain.ClaimPromptSet {
	if len(check.claims) == 0 {
		return domain.ClaimPromptSet{}
	}

	eligible := check.claims
	max := cfg.MaxJudgeCalls()
	if max > 0 && len(eligible) > max {
		slog.Warn("buildClaimPromptSet: обрезка до max_judge_calls",
			"total", len(eligible),
			"cap", max,
		)
		eligible = eligible[:max]
	}

	docLines := make(map[string][]string, len(check.structure.Docs))
	for _, doc := range check.structure.Docs {
		docLines[doc.Path] = doc.Lines
	}

	const snippetWindow = 3
	prompts := make([]domain.ClaimPrompt, 0, len(eligible))
	for _, claim := range eligible {
		prompts = append(prompts, domain.ClaimPrompt{
			DocSnippet:  contextLines(docLines[claim.File], claim.Line-1, snippetWindow),
			CodeSnippet: "",
		})
	}
	return domain.NewClaimPromptSet(prompts)
}

// contextLines возвращает строки вокруг idx (0-based) в окне ±window.
func contextLines(lines []string, idx, window int) string {
	if len(lines) == 0 {
		return ""
	}
	start := idx - window
	if start < 0 {
		start = 0
	}
	end := idx + window + 1
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

// mergeSemanticFindings конвертирует вердикты L6c в DriftFinding.
// Вердикт OK=false → finding с цитатой из судьи.
func mergeSemanticFindings(verdicts []domain.Verdict) []DriftFinding {
	var findings []DriftFinding
	for _, v := range verdicts {
		if !v.OK {
			findings = append(findings, DriftFinding{Reason: v.Quote})
		}
	}
	return findings
}

// buildDriftOutcome маппит DriftReport в LayerOutcome (L6).
// Status=fail при наличии blocker-нарушений.
func buildDriftOutcome(report DriftReport) domain.LayerOutcome {
	all := make([]DriftFinding, 0, len(report.l6a)+len(report.semantic))
	all = append(all, report.l6a...)
	all = append(all, report.semantic...)

	violations := make([]domain.Violation, 0, len(all))
	for _, f := range all {
		line := f.Claim.Line
		var linePtr *int
		if line > 0 {
			linePtr = &line
		}
		violations = append(violations, domain.Violation{
			Code:     "doc_drift",
			Layer:    "L6",
			Severity: "blocker",
			File:     filepath.ToSlash(f.Claim.File),
			Line:     linePtr,
			Message:  f.Reason,
		})
	}

	status := "pass"
	if len(violations) > 0 {
		status = "fail"
	}
	score := 100
	if len(all) > 0 {
		score = 0
	}

	return domain.LayerOutcome{
		Result: domain.LayerResult{
			Name:    "drift",
			Status:  status,
			Score:   &score,
			Summary: driftSummary(status, len(all)),
		},
		Violations: violations,
	}
}

func driftSummary(status string, count int) string {
	if status == "pass" {
		return "дрейф документации не обнаружен"
	}
	return fmt.Sprintf("дрейф документации: %d нарушение(й)", count)
}
