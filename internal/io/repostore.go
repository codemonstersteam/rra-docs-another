// Package io содержит I/O-объекты rra-docs-another: RepoStore и ReportSink.
// Каждый объект инкапсулирует свою зависимость (ФС) и возвращает доменные ошибки.
package io

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// RepoStore читает структуру репозитория с ФС.
type RepoStore struct{}

// NewRepoStore создаёт RepoStore.
func NewRepoStore() RepoStore { return RepoStore{} }

// ReadStructure обходит репозиторий и возвращает RepoStructure.
// Failure: ErrReadError (ФС недоступна).
func (s RepoStore) ReadStructure(target domain.AuditTarget) (domain.RepoStructure, error) {
	root := target.Root()

	files, mtimes, err := walkFiles(root)
	if err != nil {
		return domain.RepoStructure{}, fmt.Errorf("%w: %s", domain.ErrReadError, err)
	}

	docs, err := readMarkdownDocs(root, files)
	if err != nil {
		return domain.RepoStructure{}, fmt.Errorf("%w: %s", domain.ErrReadError, err)
	}

	manifests := collectManifests(root, files)

	return domain.RepoStructure{
		Files:     files,
		Docs:      docs,
		MTimes:    mtimes,
		Manifests: manifests,
	}, nil
}

// ReadMarkdownDocs возвращает только Markdown-документы репозитория.
func (s RepoStore) ReadMarkdownDocs(target domain.AuditTarget) ([]domain.MarkdownDoc, error) {
	root := target.Root()
	files, _, err := walkFiles(root)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrReadError, err)
	}
	docs, err := readMarkdownDocs(root, files)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrReadError, err)
	}
	return docs, nil
}

// ReadMarkdownDocsByList читает только указанные файлы (пути относительно корня репо).
// Файлы, которых нет на диске, пропускаются.
func (s RepoStore) ReadMarkdownDocsByList(target domain.AuditTarget, paths []string) ([]domain.MarkdownDoc, error) {
	root := target.Root()
	var docs []domain.MarkdownDoc
	for _, rel := range paths {
		absPath := filepath.Join(root, rel)
		doc, err := readOneMarkdown(absPath, rel)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("%w: %s", domain.ErrReadError, err)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func walkFiles(root string) ([]string, map[string]time.Time, error) {
	var files []string
	mtimes := make(map[string]time.Time)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Пропускаем скрытые директории (.git и т.п.).
			if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, rel)
		info, infoErr := d.Info()
		if infoErr == nil {
			mtimes[rel] = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return files, mtimes, nil
}

func readMarkdownDocs(root string, files []string) ([]domain.MarkdownDoc, error) {
	var docs []domain.MarkdownDoc
	for _, rel := range files {
		if !strings.HasSuffix(strings.ToLower(rel), ".md") {
			continue
		}
		doc, err := readOneMarkdown(filepath.Join(root, rel), rel)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func readOneMarkdown(absPath, rel string) (domain.MarkdownDoc, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return domain.MarkdownDoc{}, err
	}
	defer f.Close()

	var lines []string
	var headings []domain.Heading
	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		lines = append(lines, line)
		if h, ok := parseHeading(line, lineNum); ok {
			headings = append(headings, h)
		}
	}
	if err := sc.Err(); err != nil {
		return domain.MarkdownDoc{}, err
	}
	return domain.MarkdownDoc{
		Path:     rel,
		Lines:    lines,
		Headings: headings,
	}, nil
}

// parseHeading парсит строку ATX-заголовка (#...).
func parseHeading(line string, lineNum int) (domain.Heading, bool) {
	if !strings.HasPrefix(line, "#") {
		return domain.Heading{}, false
	}
	level := 0
	for _, ch := range line {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	if level < 1 || level > 6 {
		return domain.Heading{}, false
	}
	if len(line) <= level {
		return domain.Heading{}, false
	}
	// Следующий символ после # должен быть пробелом.
	if line[level] != ' ' {
		return domain.Heading{}, false
	}
	text := strings.TrimSpace(line[level+1:])
	return domain.Heading{Level: level, Text: text, Line: lineNum}, true
}

// collectManifests собирает go.mod, package.json и другие манифесты.
func collectManifests(root string, files []string) map[string]string {
	known := map[string]struct{}{
		"go.mod":           {},
		"package.json":     {},
		"Cargo.toml":       {},
		"pyproject.toml":   {},
		"requirements.txt": {},
	}
	manifests := make(map[string]string)
	for _, rel := range files {
		base := filepath.Base(rel)
		if _, ok := known[base]; !ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err == nil {
			manifests[rel] = string(data)
		}
	}
	return manifests
}
