package proxmox

import (
	"context"
	"strings"
	"testing"
)

type fakeRunner struct {
	outputs map[string]string
	errs    map[string]error
	calls   []string
}

func (f *fakeRunner) Run(_ context.Context, command string) (string, error) {
	f.calls = append(f.calls, command)
	if err, ok := f.errs[command]; ok {
		return "", err
	}
	return f.outputs[command], nil
}

func TestParseSimple(t *testing.T) {
	out := `VMID Status Name
100 running caddy
101 stopped navidrome`
	items, err := ParsePCTList(out)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if items[0].VMID != 100 || items[0].Name != "caddy" {
		t.Fatalf("unexpected first row: %+v", items[0])
	}
}

func TestMissingOptionalFields(t *testing.T) {
	out := `VMID Status Name
100 running caddy`
	items, err := ParsePCTList(out)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if items[0].PID != nil {
		t.Fatal("pid should be nil")
	}
}

func TestBlankLockColumn(t *testing.T) {
	out := `VMID Status Lock Name
100 running      caddy
101 stopped      navidrome`
	items, err := ParsePCTList(out)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if items[0].Name != "caddy" || items[1].Name != "navidrome" {
		t.Fatalf("unexpected names: %+v", items)
	}
}

func TestListLXCEnrichesConfigAndIP(t *testing.T) {
	runner := &fakeRunner{outputs: map[string]string{
		buildLXCListCommand(): `__HOMELABCTL_LIST_BEGIN__
VMID Status Lock Name
100 running      caddy
101 stopped      navidrome
__HOMELABCTL_LIST_END__
__HOMELABCTL_CONFIG_BEGIN__ 100
memory: 512
rootfs: local-lvm:vm-100-disk-0,size=8G
net0: name=eth0,bridge=vmbr0,ip=dhcp
__HOMELABCTL_CONFIG_END__ 100
__HOMELABCTL_IP_BEGIN__ 100
2: eth0    inet 10.51.51.100/24 brd 10.51.51.255 scope global eth0
__HOMELABCTL_IP_END__ 100
__HOMELABCTL_CONFIG_BEGIN__ 101
memory: 1024
rootfs: local-lvm:vm-101-disk-0,size=16G
net0: name=eth0,bridge=vmbr0,ip=10.51.51.101/24
__HOMELABCTL_CONFIG_END__ 101`,
	}}

	items, err := ListLXC(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if items[0].MemoryMB != 512 || items[0].BootdiskGB != 8 || items[0].IPAddresses[0] != "10.51.51.100" {
		t.Fatalf("unexpected enriched running container: %+v", items[0])
	}
	if items[1].MemoryMB != 1024 || items[1].BootdiskGB != 16 || items[1].IPAddresses[0] != "10.51.51.101" {
		t.Fatalf("unexpected enriched stopped container: %+v", items[1])
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 batched runner call, got %d: %#v", len(runner.calls), runner.calls)
	}
	if !strings.Contains(runner.calls[0], "pct list") || !strings.Contains(runner.calls[0], "pct config") || !strings.Contains(runner.calls[0], "pct exec") {
		t.Fatalf("unexpected batch command: %s", runner.calls[0])
	}
}

func TestParseLXCDetails(t *testing.T) {
	configs, liveIPs, err := ParseLXCDetails(`__HOMELABCTL_CONFIG_BEGIN__ 100
memory: 512
rootfs: local-lvm:vm-100-disk-0,size=8G
net0: name=eth0,bridge=vmbr0,ip=dhcp
__HOMELABCTL_CONFIG_END__ 100
__HOMELABCTL_IP_BEGIN__ 100
2: eth0    inet 10.51.51.100/24 brd 10.51.51.255 scope global eth0
__HOMELABCTL_IP_END__ 100`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if configs[100].MemoryMB != 512 || configs[100].BootdiskGB != 8 {
		t.Fatalf("unexpected config: %+v", configs[100])
	}
	if len(liveIPs[100]) != 1 || liveIPs[100][0] != "10.51.51.100" {
		t.Fatalf("unexpected live IPs: %+v", liveIPs[100])
	}
}

func TestSplitLXCListBatch(t *testing.T) {
	listOut, detailsOut, err := SplitLXCListBatch(`__HOMELABCTL_LIST_BEGIN__
VMID Status Name
100 running caddy
__HOMELABCTL_LIST_END__
__HOMELABCTL_CONFIG_BEGIN__ 100
memory: 512
__HOMELABCTL_CONFIG_END__ 100`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(listOut, "100 running caddy") {
		t.Fatalf("unexpected list output: %q", listOut)
	}
	if !strings.Contains(detailsOut, "memory: 512") {
		t.Fatalf("unexpected details output: %q", detailsOut)
	}
}

func TestParseLXCConfig(t *testing.T) {
	cfg, err := ParseLXCConfig(`memory: 2048
rootfs: local-lvm:vm-100-disk-0,size=32G
net0: name=eth0,bridge=vmbr0,ip=10.51.51.20/24
net1: name=eth1,bridge=vmbr1,ip=dhcp`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.MemoryMB != 2048 || cfg.BootdiskGB != 32 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if len(cfg.IPAddresses) != 1 || cfg.IPAddresses[0] != "10.51.51.20" {
		t.Fatalf("unexpected ips: %+v", cfg.IPAddresses)
	}
}

func TestParseIPv4Addresses(t *testing.T) {
	ips := ParseIPv4Addresses(`2: eth0    inet 10.51.51.100/24 brd 10.51.51.255 scope global eth0
4: docker0    inet 172.17.0.1/16 brd 172.17.255.255 scope global docker0
5: br-a1b2c3    inet 172.18.0.1/16 brd 172.18.255.255 scope global br-a1b2c3
3: tailscale0    inet 100.64.1.2/32 scope global tailscale0`)
	if len(ips) != 2 || ips[0] != "10.51.51.100" || ips[1] != "100.64.1.2" {
		t.Fatalf("unexpected ips: %+v", ips)
	}
}

func TestMalformedVMID(t *testing.T) {
	out := `VMID Status Name
abc running caddy`
	_, err := ParsePCTList(out)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMalformedRowIncludesDetails(t *testing.T) {
	out := `VMID Status Name
100 running`
	_, err := ParsePCTList(out)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"row 2", "got 2 fields for 3 headers", "raw row", "100 running"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error to contain %q, got %q", want, msg)
		}
	}
}

func TestEmptyOutput(t *testing.T) {
	_, err := ParsePCTList("")
	if err == nil {
		t.Fatal("expected error")
	}
}
