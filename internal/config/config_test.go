package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := writeCfg(t, `proxmox:
  host: "10.0.0.1"
  user: "root"
  private_key: "~/.ssh/id_ed25519"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Proxmox.Port != 22 {
		t.Fatalf("expected default port 22, got %d", cfg.Proxmox.Port)
	}
	if cfg.Proxmox.TimeoutSeconds != 5 {
		t.Fatalf("expected default timeout 5, got %d", cfg.Proxmox.TimeoutSeconds)
	}
}

func TestMissingHostFails(t *testing.T) {
	p := writeCfg(t, `proxmox:
  user: "root"
  private_key: "~/.ssh/id_ed25519"
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := writeCfg(t, `proxmox:
  host: "10.0.0.1"
  user: "root"
  private_key: "~/.ssh/proxmox_ed25519"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	expect := filepath.Join(home, ".ssh", "proxmox_ed25519")
	if cfg.Proxmox.PrivateKey != expect {
		t.Fatalf("expected %q, got %q", expect, cfg.Proxmox.PrivateKey)
	}
}

func writeCfg(t *testing.T, content string) string {
	t.Helper()
	d := t.TempDir()
	p := filepath.Join(d, "config.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}
