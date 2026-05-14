package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/example/go-homelabctl/internal/proxmox"
)

func TestJSONOutput(t *testing.T) {
	pid := 1234
	items := []proxmox.LXCContainer{{VMID: 100, Status: "running", Name: "caddy", MemoryMB: 512, BootdiskGB: 8, PID: &pid}}
	var b bytes.Buffer
	if err := WriteJSON(&b, items); err != nil {
		t.Fatalf("err: %v", err)
	}
	s := b.String()
	for _, part := range []string{"\"vmid\": 100", "\"name\": \"caddy\"", "\"pid\": 1234"} {
		if !strings.Contains(s, part) {
			t.Fatalf("missing %q in %s", part, s)
		}
	}
}

func TestTableOutput(t *testing.T) {
	items := []proxmox.LXCContainer{{VMID: 100, Status: "running", Name: "caddy"}}
	var b bytes.Buffer
	WriteLXCTable(&b, items)
	s := b.String()
	if !strings.Contains(s, "VMID") || !strings.Contains(s, "caddy") {
		t.Fatalf("unexpected table: %s", s)
	}
}
