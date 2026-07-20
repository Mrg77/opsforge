package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mrg77/opsforge/internal/catalog"
	"github.com/Mrg77/opsforge/internal/detect"
	"github.com/Mrg77/opsforge/internal/installer"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade every installed catalog tool through Homebrew",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !installer.Available() {
			return fmt.Errorf("homebrew is required (https://brew.sh)")
		}
		cat, err := catalog.Load()
		if err != nil {
			return err
		}
		statuses := detect.All(cat.Tools())
		upgraded, skipped, failed := 0, 0, 0
		for _, t := range cat.Tools() {
			if !statuses[t.Name].Installed {
				continue
			}
			switch res := installer.Upgrade(t); {
			case res.Err == nil:
				fmt.Printf("✓ %-16s up to date (%s)\n", t.Name, res.Backend)
				upgraded++
			case res.NotBrewManaged:
				fmt.Printf("· %-16s skipped (not installed via Homebrew)\n", t.Name)
				skipped++
			default:
				fmt.Printf("✗ %-16s %v\n%s\n", t.Name, res.Err, res.OutputTail)
				failed++
			}
		}
		fmt.Printf("\n%d checked/upgraded, %d skipped, %d failed\n", upgraded, skipped, failed)
		if failed > 0 {
			return fmt.Errorf("%d upgrade(s) failed", failed)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
