package proxmox

import "testing"

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

func TestMalformedVMID(t *testing.T) {
	out := `VMID Status Name
abc running caddy`
	_, err := ParsePCTList(out)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEmptyOutput(t *testing.T) {
	_, err := ParsePCTList("")
	if err == nil {
		t.Fatal("expected error")
	}
}
