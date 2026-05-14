package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/bartlomiejnoszka/go-homelabctl/internal/proxmox"
)

// WriteLXCTable renders LXC containers as a tab-aligned text table.
// This is presentation code; it does not know how containers were fetched.
func WriteLXCTable(w io.Writer, items []proxmox.LXCContainer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "VMID\tSTATUS\tNAME\tMEMORY\tBOOTDISK\tFREE\tUSE%\tIP\tPID")
	for _, c := range items {
		pid := "-"
		if c.PID != nil {
			pid = fmt.Sprintf("%d", *c.PID)
		}
		ip := "-"
		if len(c.IPAddresses) > 0 {
			ip = strings.Join(c.IPAddresses, ",")
		}
		free := "-"
		if c.BootdiskFreeGB != nil {
			free = fmt.Sprintf("%.2f", *c.BootdiskFreeGB)
		}
		usedPercent := "-"
		if c.BootdiskUsedPercent != nil {
			usedPercent = fmt.Sprintf("%.0f%%", *c.BootdiskUsedPercent)
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%.2f\t%s\t%s\t%s\t%s\n", c.VMID, c.Status, c.Name, c.MemoryMB, c.BootdiskGB, free, usedPercent, ip, pid)
	}
	_ = tw.Flush()
}
