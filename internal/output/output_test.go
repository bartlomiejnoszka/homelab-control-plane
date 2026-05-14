package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bartlomiejnoszka/go-homelabctl/internal/proxmox"
)

func TestJSONOutput(t *testing.T) {
	pid := 1234
	freeGB := 6.5
	usedPercent := 19.0
	items := []proxmox.LXCContainer{{VMID: 100, Status: "running", Name: "caddy", MemoryMB: 512, BootdiskGB: 8, BootdiskFreeGB: &freeGB, BootdiskUsedPercent: &usedPercent, IPAddresses: []string{"10.51.51.100"}, PID: &pid}}
	var b bytes.Buffer
	if err := WriteJSON(&b, items); err != nil {
		t.Fatalf("err: %v", err)
	}
	s := b.String()
	for _, part := range []string{"\"vmid\": 100", "\"name\": \"caddy\"", "\"bootdisk_free_gb\": 6.5", "\"bootdisk_used_percent\": 19", "\"ip_addresses\": [", "\"10.51.51.100\"", "\"pid\": 1234"} {
		if !strings.Contains(s, part) {
			t.Fatalf("missing %q in %s", part, s)
		}
	}
}

func TestTableOutput(t *testing.T) {
	freeGB := 6.5
	usedPercent := 19.0
	items := []proxmox.LXCContainer{{VMID: 100, Status: "running", Name: "caddy", BootdiskFreeGB: &freeGB, BootdiskUsedPercent: &usedPercent, IPAddresses: []string{"10.51.51.100"}}}
	var b bytes.Buffer
	WriteLXCTable(&b, items)
	s := b.String()
	if !strings.Contains(s, "VMID") || !strings.Contains(s, "FREE") || !strings.Contains(s, "USE%") || !strings.Contains(s, "19%") || !strings.Contains(s, "caddy") || !strings.Contains(s, "10.51.51.100") {
		t.Fatalf("unexpected table: %s", s)
	}
}
