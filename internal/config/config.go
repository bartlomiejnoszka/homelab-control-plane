package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPort    = 22
	defaultTimeout = 5 * time.Second
)

type Config struct {
	Proxmox ProxmoxConfig `yaml:"proxmox"`
}

type ProxmoxConfig struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	User           string        `yaml:"user"`
	PrivateKey     string        `yaml:"private_key"`
	TimeoutSeconds int           `yaml:"timeout_seconds"`
	Timeout        time.Duration `yaml:"-"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "homelabctl", "config.yaml"), nil
}

func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("config path is required")
	}

	expandedPath, err := expandTilde(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(expandedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %s", expandedPath)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	if err := cfg.applyDefaultsAndValidate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyDefaultsAndValidate() error {
	p := &c.Proxmox
	p.Host = strings.TrimSpace(p.Host)
	p.User = strings.TrimSpace(p.User)
	p.PrivateKey = strings.TrimSpace(p.PrivateKey)

	if p.Port == 0 {
		p.Port = defaultPort
	}
	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = int(defaultTimeout.Seconds())
	}
	p.Timeout = time.Duration(p.TimeoutSeconds) * time.Second

	if p.Host == "" {
		return errors.New("invalid config: proxmox.host is required")
	}
	if p.User == "" {
		return errors.New("invalid config: proxmox.user is required")
	}
	if p.PrivateKey == "" {
		return errors.New("invalid config: proxmox.private_key is required")
	}

	expandedKey, err := expandTilde(p.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	p.PrivateKey = expandedKey
	return nil
}

func expandTilde(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
