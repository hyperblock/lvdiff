package main

import (
	"os"

	"fmt"

	"strings"

	"lvbackup"

	"github.com/spf13/cobra"
)

type Pair struct {
	key, value string
}

const c_HEADER = "HYPERLAYER/1.0\n"

func main() {
	var rootCmd *cobra.Command
	var pool string
	//	var head string
	var strPair []string
	//	var value []string
	var vgname string
	var vol, backingVol string
	var depth int32
	//var output string
	header := c_HEADER

	rootCmd = &cobra.Command{
		Use:   "lvdiff <volume> <backing-volume>",
		Short: "lvdiff is a tool to backup LVM2 thinly-provisioned volumes, will dump the thin volume $volume's incremental block from $backing-volume",
		Run: func(cmd *cobra.Command, args []string) {
			if pool == "" || vgname == "" || len(args) < 2 {
				fmt.Fprintf(os.Stderr, "Too few arguments.")
				rootCmd.Usage()
				return
			}
			if depth < 0 && depth > 3 {
				fmt.Fprintf(os.Stderr, "Detect level range: 0-3")
				rootCmd.Usage()
				return
			}
			pair := []Pair{}
			for _, obj := range strPair {
				token := strings.Split(obj, ":")
				if len(token) != 2 {
					fmt.Fprintf(os.Stderr, "Invalid key-value pair: %s\n", obj)
					rootCmd.Usage()
					return
				}
				token[0] = strings.TrimLeft(strings.TrimRight(token[0], " "), " ")
				token[1] = strings.TrimLeft(strings.TrimRight(token[1], " "), " ")
				pair = append(pair, Pair{key: token[0], value: token[1]})
			}
			vol, backingVol = args[0], args[1]
			f := os.Stdout

			sender, err := lvbackup.NewStreamSender(vgname, vol, backingVol, f, int(depth))

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if err := sender.Run(header); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
		},
	}
	rootCmd.Flags().StringVarP(&vgname, "lvgroup", "g", "", "volume group.")
	rootCmd.Flags().StringVarP(&pool, "pool", "p", "", "thin volume pool.")
	rootCmd.Flags().Int32VarP(&depth, "", "d", 2, `checksum detect level. range: 0-3 
														0 means no checksum, 
														1 means only check head block, 
														2 means random check, 
														3 means scan all data blocks.`)
	//	rootCmd.Flags().StringVarP(&vol, "vol", "v", "", "thin volume name.")
	//	rootCmd.Flags().StringVarP(&backingVol, "backing-volume", "b", "", "thin volume name.")
	rootCmd.Flags().StringArrayVarP(&strPair, "pair", "", nil, "set key-value pair (format as '$key:$value').")
	//rootCmd.Flags().StringArrayVarP(&value, "value", "", nil, "set value.")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}

	os.Exit(0)
}
