package gen

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	var output string
	var input string

	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate GORM query code from raw SQL interfaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			g := Generator{
				Name:  filepath.Base(output),
				Files: map[string]*File{},
			}

			err := g.Process(input, output)
			if err != nil {
				return fmt.Errorf("error processing %s: %v", input, err)
			}

			err = g.Gen()
			if err != nil {
				return fmt.Errorf("error render template got error: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "./g", "Directory to place generated code")
	cmd.Flags().StringVarP(&input, "input", "i", "", "Path to Go interface file with raw SQL annotations")
	cmd.MarkFlagRequired("input")

	return cmd
}
