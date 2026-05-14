package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/example/go-homelabctl/internal/proxmox"
)

func TestJSONOutput(t *testing.T) {
	pid := 1234
	items := []proxmox.LXCContainer{{VMID: 100, Status: "running", Name: "caddy", MemoryMB: 512, BootdiskGB: 8, IPAddresses: []string{"10.51.51.100"}, PID: &pid}}
	var b bytes.Buffer
	if err := WriteJSON(&b, items); err != nil {
		t.Fatalf("err: %v", err)
	}
	s := b.String()
	for _, part := range []string{"\"vmid\": 100", "\"name\": \"caddy\"", "\"ip_addresses\": [", "\"10.51.51.100\"", "\"pid\": 1234"} {
		if !strings.Contains(s, part) {
			t.Fatalf("missing %q in %s", part, s)
		}
	}
}

func TestTableOutput(t *testing.T) {
	items := []proxmox.LXCContainer{{VMID: 100, Status: "running", Name: "caddy", IPAddresses: []string{"10.51.51.100"}}}
	var b bytes.Buffer
	WriteLXCTable(&b, items)
	s := b.String()
	if !strings.Contains(s, "VMID") || !strings.Contains(s, "IP") || !strings.Contains(s, "caddy") || !strings.Contains(s, "10.51.51.100") {
		t.Fatalf("unexpected table: %s", s)
	}
}
