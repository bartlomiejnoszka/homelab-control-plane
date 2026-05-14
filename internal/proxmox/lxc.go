package proxmox

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, command string) (string, error)
}

type LXCContainer struct {
	VMID                int      `json:"vmid"`
	Status              string   `json:"status"`
	Name                string   `json:"name"`
	MemoryMB            int      `json:"memory_mb,omitempty"`
	BootdiskGB          float64  `json:"bootdisk_gb,omitempty"`
	BootdiskFreeGB      *float64 `json:"bootdisk_free_gb,omitempty"`
	BootdiskUsedPercent *float64 `json:"bootdisk_used_percent,omitempty"`
	IPAddresses         []string `json:"ip_addresses,omitempty"`
	PID                 *int     `json:"pid,omitempty"`
}

type LXCConfig struct {
	MemoryMB    int
	BootdiskGB  float64
	IPAddresses []string
}

type LXCDiskUsage struct {
	FreeGB      float64
	UsedPercent float64
}

const (
	listBeginMarker   = "__HOMELABCTL_LIST_BEGIN__"
	listEndMarker     = "__HOMELABCTL_LIST_END__"
	configBeginMarker = "__HOMELABCTL_CONFIG_BEGIN__"
	configEndMarker   = "__HOMELABCTL_CONFIG_END__"
	ipBeginMarker     = "__HOMELABCTL_IP_BEGIN__"
	ipEndMarker       = "__HOMELABCTL_IP_END__"
	diskBeginMarker   = "__HOMELABCTL_DISK_BEGIN__"
	diskEndMarker     = "__HOMELABCTL_DISK_END__"
)

func ListLXC(ctx context.Context, runner Runner) ([]LXCContainer, error) {
	stdout, err := runner.Run(ctx, buildLXCListCommand())
	if err != nil {
		return nil, err
	}
	listOut, detailsOut, err := SplitLXCListBatch(stdout)
	if err != nil {
		return nil, err
	}
	containers, err := ParsePCTList(listOut)
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return containers, nil
	}

	configs, liveIPs, diskUsages, err := ParseLXCDetails(detailsOut)
	if err != nil {
		return nil, err
	}
	for i := range containers {
		cfg, ok := configs[containers[i].VMID]
		if !ok {
			return nil, fmt.Errorf("details missing config for VMID %d", containers[i].VMID)
		}
		containers[i].MemoryMB = cfg.MemoryMB
		containers[i].BootdiskGB = cfg.BootdiskGB
		containers[i].IPAddresses = cfg.IPAddresses

		if ips := liveIPs[containers[i].VMID]; len(ips) > 0 {
			containers[i].IPAddresses = ips
		}
		if usage, ok := diskUsages[containers[i].VMID]; ok {
			containers[i].BootdiskFreeGB = &usage.FreeGB
			containers[i].BootdiskUsedPercent = &usage.UsedPercent
		}
	}
	return containers, nil
}

func buildLXCListCommand() string {
	return fmt.Sprintf(`set -e
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
list="$(pct list)"
echo %s
printf '%%s\n' "$list"
echo %s
ids="$tmpdir/ids"
printf '%%s\n' "$list" | awk 'NR > 1 { print $1 " " $2 }' > "$ids"
while read -r vmid status; do
  [ -n "$vmid" ] || continue
  (
    (
      set -e
      {
        echo %s "$vmid"
        config="$(pct config "$vmid")"
        printf '%%s\n' "$config"
        echo %s "$vmid"
        if [ "$status" = "running" ]; then
          echo %s "$vmid"
          pct exec "$vmid" -- df -B1 -P / 2>/dev/null || true
          echo %s "$vmid"
        fi
        if [ "$status" = "running" ] && ! printf '%%s\n' "$config" | grep -Eq '(^|[,[:space:]])ip=[0-9]+\.'; then
          echo %s "$vmid"
          pct exec "$vmid" -- ip -4 -o addr show scope global 2>/dev/null || true
          echo %s "$vmid"
        fi
      } > "$tmpdir/$vmid.out"
    ) || touch "$tmpdir/$vmid.err"
  ) &
done < "$ids"
wait
if ls "$tmpdir"/*.err >/dev/null 2>&1; then
  exit 1
fi
while read -r vmid status; do
  [ -n "$vmid" ] || continue
  cat "$tmpdir/$vmid.out"
done < "$ids"
`, listBeginMarker, listEndMarker, configBeginMarker, configEndMarker, diskBeginMarker, diskEndMarker, ipBeginMarker, ipEndMarker)
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
		fields, err := alignFields(headers, strings.Fields(line))
		if err != nil {
			return nil, fmt.Errorf("parser failed on row %d: %w; raw row: %q", li+2, err, line)
		}
		c, err := parseRow(fields, idx)
		if err != nil {
			return nil, fmt.Errorf("parser failed on row %d: %w; raw row: %q", li+2, err, line)
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func alignFields(headers, fields []string) ([]string, error) {
	if len(fields) >= len(headers) {
		return fields, nil
	}

	lockIdx := headerIndex(headers, "lock")
	if lockIdx >= 0 && len(fields) == len(headers)-1 {
		aligned := make([]string, 0, len(headers))
		aligned = append(aligned, fields[:lockIdx]...)
		aligned = append(aligned, "")
		aligned = append(aligned, fields[lockIdx:]...)
		return aligned, nil
	}

	return nil, fmt.Errorf("malformed row: got %d fields for %d headers %v", len(fields), len(headers), headers)
}

func headerIndex(headers []string, name string) int {
	for i, h := range headers {
		if strings.EqualFold(h, name) {
			return i
		}
	}
	return -1
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

func SplitLXCListBatch(output string) (string, string, error) {
	var listLines []string
	var detailLines []string
	inList := false
	foundList := false

	for _, line := range strings.Split(output, "\n") {
		switch strings.TrimSpace(line) {
		case listBeginMarker:
			inList = true
			foundList = true
			continue
		case listEndMarker:
			inList = false
			continue
		}

		if inList {
			listLines = append(listLines, line)
		} else if strings.TrimSpace(line) != "" {
			detailLines = append(detailLines, line)
		}
	}

	if !foundList {
		return "", "", fmt.Errorf("batch output missing %s marker", listBeginMarker)
	}
	return strings.Join(listLines, "\n"), strings.Join(detailLines, "\n"), nil
}

func ParseLXCDetails(output string) (map[int]LXCConfig, map[int][]string, map[int]LXCDiskUsage, error) {
	rawConfigs := map[int][]string{}
	rawIPs := map[int][]string{}
	rawDisks := map[int][]string{}

	section := ""
	vmid := 0
	for _, line := range strings.Split(output, "\n") {
		marker, markerVMID, ok, err := parseDetailsMarker(line)
		if err != nil {
			return nil, nil, nil, err
		}
		if ok {
			switch marker {
			case configBeginMarker:
				section, vmid = "config", markerVMID
			case ipBeginMarker:
				section, vmid = "ip", markerVMID
			case diskBeginMarker:
				section, vmid = "disk", markerVMID
			case configEndMarker, ipEndMarker, diskEndMarker:
				section, vmid = "", 0
			}
			continue
		}

		switch section {
		case "config":
			rawConfigs[vmid] = append(rawConfigs[vmid], line)
		case "ip":
			rawIPs[vmid] = append(rawIPs[vmid], line)
		case "disk":
			rawDisks[vmid] = append(rawDisks[vmid], line)
		}
	}

	configs := make(map[int]LXCConfig, len(rawConfigs))
	for vmid, lines := range rawConfigs {
		cfg, err := ParseLXCConfig(strings.Join(lines, "\n"))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse config for VMID %d: %w", vmid, err)
		}
		configs[vmid] = cfg
	}

	liveIPs := make(map[int][]string, len(rawIPs))
	for vmid, lines := range rawIPs {
		liveIPs[vmid] = ParseIPv4Addresses(strings.Join(lines, "\n"))
	}

	diskUsages := make(map[int]LXCDiskUsage, len(rawDisks))
	for vmid, lines := range rawDisks {
		usage, err := ParseDiskUsage(strings.Join(lines, "\n"))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse disk usage for VMID %d: %w", vmid, err)
		}
		diskUsages[vmid] = usage
	}

	return configs, liveIPs, diskUsages, nil
}

func parseDetailsMarker(line string) (string, int, bool, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "__HOMELABCTL_") {
		return "", 0, false, nil
	}
	if len(fields) != 2 {
		return "", 0, true, fmt.Errorf("invalid details marker %q", line)
	}
	vmid, err := strconv.Atoi(fields[1])
	if err != nil {
		return "", 0, true, fmt.Errorf("invalid details marker VMID %q", fields[1])
	}
	switch fields[0] {
	case configBeginMarker, configEndMarker, ipBeginMarker, ipEndMarker, diskBeginMarker, diskEndMarker:
		return fields[0], vmid, true, nil
	default:
		return "", 0, true, fmt.Errorf("unknown details marker %q", fields[0])
	}
}

func ParseDiskUsage(output string) (LXCDiskUsage, error) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "Filesystem" {
			continue
		}
		availableBytes, err := strconv.ParseFloat(fields[3], 64)
		if err != nil {
			return LXCDiskUsage{}, fmt.Errorf("invalid available bytes %q", fields[3])
		}
		usedPercent, err := strconv.ParseFloat(strings.TrimSuffix(fields[4], "%"), 64)
		if err != nil {
			return LXCDiskUsage{}, fmt.Errorf("invalid used percent %q", fields[4])
		}
		freeGB := availableBytes / (1024 * 1024 * 1024)
		return LXCDiskUsage{FreeGB: math.Round(freeGB*100) / 100, UsedPercent: usedPercent}, nil
	}
	return LXCDiskUsage{}, fmt.Errorf("disk usage not found")
}

func ParseLXCConfig(output string) (LXCConfig, error) {
	var cfg LXCConfig
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case key == "memory":
			mem, err := strconv.Atoi(value)
			if err != nil {
				return LXCConfig{}, fmt.Errorf("invalid memory value %q", value)
			}
			cfg.MemoryMB = mem
		case key == "rootfs":
			size, err := rootfsSizeGB(value)
			if err != nil {
				return LXCConfig{}, err
			}
			cfg.BootdiskGB = size
		case strings.HasPrefix(key, "net"):
			cfg.IPAddresses = append(cfg.IPAddresses, staticIPv4FromNet(value)...)
		}
	}
	return cfg, nil
}

func rootfsSizeGB(value string) (float64, error) {
	for _, part := range strings.Split(value, ",") {
		key, rawSize, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || key != "size" {
			continue
		}
		size, err := parseSizeGB(rawSize)
		if err != nil {
			return 0, fmt.Errorf("invalid rootfs size %q", rawSize)
		}
		return size, nil
	}
	return 0, nil
}

func parseSizeGB(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty size")
	}

	multiplier := 1.0 / (1024 * 1024 * 1024)
	unit := raw[len(raw)-1]
	switch unit {
	case 'K', 'k':
		multiplier = 1.0 / (1024 * 1024)
		raw = raw[:len(raw)-1]
	case 'M', 'm':
		multiplier = 1.0 / 1024
		raw = raw[:len(raw)-1]
	case 'G', 'g':
		multiplier = 1
		raw = raw[:len(raw)-1]
	case 'T', 't':
		multiplier = 1024
		raw = raw[:len(raw)-1]
	}

	size, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return size * multiplier, nil
}

func staticIPv4FromNet(value string) []string {
	var ips []string
	for _, part := range strings.Split(value, ",") {
		key, rawIP, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || key != "ip" || rawIP == "dhcp" || rawIP == "manual" {
			continue
		}
		ip, _, _ := strings.Cut(rawIP, "/")
		if strings.Contains(ip, ".") {
			ips = append(ips, ip)
		}
	}
	return ips
}

func ParseIPv4Addresses(output string) []string {
	var ips []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || ignoredInterface(fields[1]) {
			continue
		}
		for i, field := range fields {
			if field == "inet" && i+1 < len(fields) {
				ip, _, _ := strings.Cut(fields[i+1], "/")
				if strings.Contains(ip, ".") {
					ips = append(ips, ip)
				}
			}
		}
	}
	return ips
}

func ignoredInterface(name string) bool {
	name = strings.TrimSuffix(name, ":")
	return name == "lo" ||
		name == "docker0" ||
		strings.HasPrefix(name, "br-") ||
		strings.HasPrefix(name, "veth")
}
