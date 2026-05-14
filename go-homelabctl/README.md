# homelabctl

`homelabctl` is a small Go CLI for managing a Proxmox homelab over SSH.

## Purpose

First MVP command: `homelabctl lxc list`

It loads YAML config, runs `pct list` on Proxmox via SSH, parses results, and renders table or JSON.

## Install

```bash
go mod tidy
go build -o homelabctl ./cmd/homelabctl
```

## Configuration

Default path: `~/.config/homelabctl/config.yaml`

```yaml
proxmox:
  host: "10.51.51.2"
  port: 22
  user: "root"
  private_key: "~/.ssh/proxmox_ed25519"
  timeout_seconds: 5
```

## Commands

```bash
go run ./cmd/homelabctl --help
go run ./cmd/homelabctl lxc --help
go run ./cmd/homelabctl --config ./config.yaml lxc list
go run ./cmd/homelabctl --config ./config.yaml lxc list --json
```

## Example output

Table:

```text
VMID   STATUS    NAME          MEMORY   BOOTDISK   PID
100    running   caddy         512      8.00       1234
101    stopped   navidrome     1024     16.00      -
```

JSON:

```json
[
  {
    "vmid": 100,
    "status": "running",
    "name": "caddy",
    "memory_mb": 512,
    "bootdisk_gb": 8,
    "pid": 1234
  }
]
```

## Development

```bash
go mod tidy
go run ./cmd/homelabctl --help
go run ./cmd/homelabctl --config ./config.yaml lxc list
go test ./...
```
