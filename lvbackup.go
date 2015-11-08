package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func setupSendCommand(rootCmd *cobra.Command) {
	var vgname, lvname, srcname string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "create a stream representation of thin LV into standard output",
		Run: func(cmd *cobra.Command, args []string) {
			if len(vgname) == 0 || len(lvname) == 1 {
				fmt.Fprintln(os.Stderr, "volume group and logical volume must be provided")
				cmd.Usage()
				os.Exit(1)
			}

			sender, err := newStreamSender(vgname, lvname, srcname, os.Stdout)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}

			if err := sender.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(3)
			}

			os.Exit(0)
		},
	}

	cmd.Flags().StringVarP(&vgname, "vg", "v", "", "volume group")
	cmd.Flags().StringVarP(&lvname, "lv", "l", "", "logical volume")
	cmd.Flags().StringVarP(&srcname, "incremental", "i", "", "source logical volume")

	rootCmd.AddCommand(cmd)
}

func setupRecvCommand(rootCmd *cobra.Command) {
	var vgname, poolname, lvname string

	cmd := &cobra.Command{
		Use:   "recv",
		Short: "create or update thin logcial volume with contents in standard input",
		Run: func(cmd *cobra.Command, args []string) {
			if len(vgname) == 0 || len(poolname) == 0 || len(lvname) == 0 {
				fmt.Fprintln(os.Stderr, "volume group, thin pool and logical volume must be provided")
				cmd.Usage()
				os.Exit(-1)
			}

			recver, err := newStreamRecver(vgname, poolname, lvname, os.Stdin)
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

	cmd.Flags().StringVarP(&vgname, "vg", "v", "", "volume group")
	cmd.Flags().StringVarP(&poolname, "pool", "p", "", "thin pool")
	cmd.Flags().StringVarP(&lvname, "lv", "l", "", "logical volume")

	rootCmd.AddCommand(cmd)
}

func setupInfoCommand(rootCmd *cobra.Command) {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "info [-v] DATA_FILE",
		Short: "show info of stream data file created by send sub command",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Usage()
				os.Exit(-1)
			}

			if err := showInfo(args[0], verbose); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show verbose info")

	rootCmd.AddCommand(cmd)
}

func setupMergeCommand(rootCmd *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "merge multiple incremental backup into single backup",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, "not implemented yet")
			os.Exit(-1)
		},
	}

	rootCmd.AddCommand(cmd)
}

func main() {
	var rootCmd *cobra.Command

	rootCmd = &cobra.Command{
		Use:   "lvbackup",
		Short: "lvbackup is a tool to backup and restore LVM2 thinly-provisioned volumes",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.Usage()
			os.Exit(-1)
		},
	}

	setupSendCommand(rootCmd)
	setupRecvCommand(rootCmd)
	setupInfoCommand(rootCmd)
	setupMergeCommand(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}

	os.Exit(0)
}
