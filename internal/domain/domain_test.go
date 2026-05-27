package domain_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemonstersteam/rra-docs-another/internal/domain"
)

// ── NewAuditTarget ───────────────────────────────────────────────────────────

func TestNewAuditTarget_happy(t *testing.T) {
	dir := t.TempDir()
	req := domain.Request{Path: dir}
	got, err := domain.NewAuditTarget(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Root() == "" {
		t.Error("Root must not be empty")
	}
}

func TestNewAuditTarget_pathNotFound(t *testing.T) {
	req := domain.Request{Path: "/no-such-dir-zzzzzz"}
	_, err := domain.NewAuditTarget(req)
	if !errors.Is(err, domain.ErrPathNotFound) {
		t.Fatalf("expected ErrPathNotFound, got %v", err)
	}
}

func TestNewAuditTarget_noPerms(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Skip("cannot create unreadable dir:", err)
	}
	defer os.Chmod(locked, 0o700) //nolint:errcheck

	if os.Getuid() == 0 {
		t.Skip("root ignores permission bits")
	}
	req := domain.Request{Path: locked}
	_, err := domain.NewAuditTarget(req)
	if !errors.Is(err, domain.ErrReadError) {
		t.Fatalf("expected ErrReadError, got %v", err)
	}
}

// ── NewConfig ────────────────────────────────────────────────────────────────

func TestNewConfig_happy(t *testing.T) {
	req := domain.Request{}
	cfg, err := domain.NewConfig(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DriftThresholdDays() <= 0 {
		t.Error("DriftThresholdDays must be positive")
	}
}

func TestNewConfig_badConfig(t *testing.T) {
	req := domain.Request{ConfigPath: "/no-such-config-file-zzzz.yaml"}
	_, err := domain.NewConfig(req)
	if !errors.Is(err, domain.ErrConfigInvalid) {
		t.Fatalf("expected ErrConfigInvalid, got %v", err)
	}
}
