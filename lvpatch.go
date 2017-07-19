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
	var vgname, baseLv, newLv string

	rootCmd = &cobra.Command{
		Use:   "lvpatch <new_volume_name>",
		Short: "create or update thin logcial volume with contents in standard input",
		Run: func(cmd *cobra.Command, args []string) {
			if len(vgname) == 0 || len(baseLv) == 0 {
				fmt.Fprintln(os.Stderr, "volume group, thin pool and logical volume must be provided")
				cmd.Usage()
				os.Exit(-1)
			}
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "too few arguments.")
				cmd.Usage()
				os.Exit(-1)
			}
			newLv = args[0]
			recver, err := lvbackup.NewStreamRecver(vgname, "", baseLv, flg, os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}

			if err := recver.Run(newLv); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(3)
			}

			os.Exit(0)
		},
	}

	rootCmd.Flags().StringVarP(&vgname, "lvgroup", "g", "", "volume group")
	rootCmd.Flags().BoolVarP(&flg, "no-base-check", "", false, "patch volume without check blocks' hash.")
	//rootCmd.Flags().StringVarP(&poolname, "pool", "p", "", "thin pool")
	rootCmd.Flags().StringVarP(&baseLv, "lvbase", "l", "", "base logical volume")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}

	os.Exit(0)
}
