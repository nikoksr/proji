package cmd

import (
	"fmt"

	"github.com/nikoksr/proji/internal/app/proji/class"
	"github.com/spf13/cobra"
)

var classImportCmd = &cobra.Command{
	Use:   "import FILE",
	Short: "Import a proji class from a config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("Missing configfile name")
		}

		for _, configName := range args {
			err := class.Import(configName)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	classCmd.AddCommand(classImportCmd)
}