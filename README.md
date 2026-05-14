# homelabctl

`homelabctl` is a small Go CLI for managing a Proxmox homelab over SSH.

## Purpose

First MVP command: `homelabctl lxc list`

It loads YAML config, runs Proxmox `pct` commands over SSH, enriches the LXC list with memory, root disk size, free root disk space, usage percentage, and IP addresses, then renders table or JSON.

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
VMID   STATUS    NAME          MEMORY   BOOTDISK   FREE   USE%   IP            PID
100    running   caddy         512      8.00       6.50   19%    10.51.51.100  1234
101    stopped   navidrome     1024     16.00      -      -      10.51.51.101  -
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
    "bootdisk_free_gb": 6.5,
    "bootdisk_used_percent": 19,
    "ip_addresses": [
      "10.51.51.100"
    ],
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

## Learning Go

For a PHP/Symfony-oriented walkthrough of the codebase, see [docs/go-for-php-developers.md](docs/go-for-php-developers.md).
