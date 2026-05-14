package proxmox

import (
	"fmt"
	"strconv"
	"strings"
)

type LXCContainer struct {
	VMID       int     `json:"vmid"`
	Status     string  `json:"status"`
	Name       string  `json:"name"`
	MemoryMB   int     `json:"memory_mb,omitempty"`
	BootdiskGB float64 `json:"bootdisk_gb,omitempty"`
	PID        *int    `json:"pid,omitempty"`
}

func ParsePCTList(output string) ([]LXCContainer, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, fmt.Errorf("parser failed: empty output")
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("parser failed: missing rows")
	}

	headers := strings.Fields(lines[0])
	idx := map[string]int{}
	for i, h := range headers {
		idx[strings.ToLower(h)] = i
	}
	for _, req := range []string{"vmid", "status", "name"} {
		if _, ok := idx[req]; !ok {
			return nil, fmt.Errorf("parser failed: required column %q not found", req)
		}
	}

	containers := make([]LXCContainer, 0, len(lines)-1)
	for li, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < len(headers) {
			return nil, fmt.Errorf("parser failed: malformed row %d", li+2)
		}
		c, err := parseRow(fields, idx)
		if err != nil {
			return nil, fmt.Errorf("parser failed on row %d: %w", li+2, err)
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func parseRow(fields []string, idx map[string]int) (LXCContainer, error) {
	vmid, err := strconv.Atoi(fields[idx["vmid"]])
	if err != nil {
		return LXCContainer{}, fmt.Errorf("invalid VMID %q", fields[idx["vmid"]])
	}
	c := LXCContainer{VMID: vmid, Status: fields[idx["status"]], Name: fields[idx["name"]]}

	if i, ok := idx["mem"]; ok && i < len(fields) {
		if mem, err := strconv.Atoi(fields[i]); err == nil {
			c.MemoryMB = mem
		}
	}
	if i, ok := idx["memory"]; ok && i < len(fields) {
		if mem, err := strconv.Atoi(fields[i]); err == nil {
			c.MemoryMB = mem
		}
	}
	if i, ok := idx["bootdisk"]; ok && i < len(fields) {
		if b, err := strconv.ParseFloat(fields[i], 64); err == nil {
			c.BootdiskGB = b
		}
	}
	if i, ok := idx["pid"]; ok && i < len(fields) && fields[i] != "-" {
		if p, err := strconv.Atoi(fields[i]); err == nil {
			c.PID = &p
		}
	}

	return c, nil
}
