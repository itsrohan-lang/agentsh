package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const defaultConfig = `version: 1
config:
  # Advisory only: kernel egress enforcement is currently unimplemented.
  enforcement_mode: warn
  network:
    allow: []
    deny: []
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Writes a .agentsh.yaml config file in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		const path = ".agentsh.yaml"

		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; remove it first to regenerate", path)
		}
		if _, err := os.Stat(".stuntdouble.yaml"); err == nil {
			return fmt.Errorf(".stuntdouble.yaml already exists; remove it first to regenerate", )
		}

		if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}

		fmt.Printf("✅ Wrote %s\n", path)
		fmt.Println("Note: network egress filtering is advisory until kernel eBPF is attached.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
