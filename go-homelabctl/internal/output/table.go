package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/example/go-homelabctl/internal/proxmox"
)

func WriteLXCTable(w io.Writer, items []proxmox.LXCContainer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "VMID\tSTATUS\tNAME\tMEMORY\tBOOTDISK\tPID")
	for _, c := range items {
		pid := "-"
		if c.PID != nil {
			pid = fmt.Sprintf("%d", *c.PID)
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%.2f\t%s\n", c.VMID, c.Status, c.Name, c.MemoryMB, c.BootdiskGB, pid)
	}
	_ = tw.Flush()
}
