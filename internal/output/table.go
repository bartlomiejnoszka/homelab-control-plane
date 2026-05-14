package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/example/go-homelabctl/internal/proxmox"
)

func WriteLXCTable(w io.Writer, items []proxmox.LXCContainer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "VMID\tSTATUS\tNAME\tMEMORY\tBOOTDISK\tIP\tPID")
	for _, c := range items {
		pid := "-"
		if c.PID != nil {
			pid = fmt.Sprintf("%d", *c.PID)
		}
		ip := "-"
		if len(c.IPAddresses) > 0 {
			ip = strings.Join(c.IPAddresses, ",")
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%.2f\t%s\t%s\n", c.VMID, c.Status, c.Name, c.MemoryMB, c.BootdiskGB, ip, pid)
	}
	_ = tw.Flush()
}
