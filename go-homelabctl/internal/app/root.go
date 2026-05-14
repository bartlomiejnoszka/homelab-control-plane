package app

import (
	"context"
	"os"

	"github.com/example/go-homelabctl/internal/config"
	"github.com/example/go-homelabctl/internal/output"
	"github.com/example/go-homelabctl/internal/proxmox"
	"github.com/example/go-homelabctl/internal/sshx"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var configPath string
	defPath, _ := config.DefaultPath()

	rootCmd := &cobra.Command{
		Use:   "homelabctl",
		Short: "Manage homelab workloads on Proxmox",
	}
	rootCmd.PersistentFlags().StringVar(&configPath, "config", defPath, "Path to config file")

	lxcCmd := &cobra.Command{Use: "lxc", Short: "LXC-related commands"}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Proxmox LXC containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOut, _ := cmd.Flags().GetBool("json")
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			runner := sshx.NewSSHRunner(cfg.Proxmox.Host, cfg.Proxmox.Port, cfg.Proxmox.User, cfg.Proxmox.PrivateKey, cfg.Proxmox.Timeout)
			stdout, err := runner.Run(context.Background(), "pct list")
			if err != nil {
				return err
			}
			containers, err := proxmox.ParsePCTList(stdout)
			if err != nil {
				return err
			}
			if jsonOut {
				return output.WriteJSON(os.Stdout, containers)
			}
			output.WriteLXCTable(os.Stdout, containers)
			return nil
		},
	}
	listCmd.Flags().Bool("json", false, "Output as JSON")
	lxcCmd.AddCommand(listCmd)
	rootCmd.AddCommand(lxcCmd)

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	return rootCmd
}
