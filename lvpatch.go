package main

import (
	"os"

	"github.com/hyperblock/lvdiff/lvbackup"

	"fmt"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd *cobra.Command
	var flg bool
	var vgname, poolname, lvname, input string

	rootCmd = &cobra.Command{
		Use:   "lvpatch <input_diff_file>",
		Short: "create or update thin logcial volume with contents in standard input",
		Run: func(cmd *cobra.Command, args []string) {
			if len(vgname) == 0 || len(poolname) == 0 || len(lvname) == 0 || len(args) == 0 {
				fmt.Fprintln(os.Stderr, "volume group, thin pool and logical volume must be provided")
				cmd.Usage()
				os.Exit(-1)
			}
			input = args[0]
			f, err := os.Open(input)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer f.Close()
			recver, err := lvbackup.NewStreamRecver(vgname, poolname, lvname, flg, f)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}

			if err := recver.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(3)
			}

			os.Exit(0)
		},
	}

	rootCmd.Flags().StringVarP(&vgname, "lvgroup", "g", "", "volume group")
	rootCmd.Flags().BoolVarP(&flg, "no-base-check", "", false, "patch volume without check blocks' hash.")
	rootCmd.Flags().StringVarP(&poolname, "pool", "p", "", "thin pool")
	rootCmd.Flags().StringVarP(&lvname, "lv", "l", "", "logical volume")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}

	os.Exit(0)
}
