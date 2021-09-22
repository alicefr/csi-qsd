package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Port string
var Host string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "qsd-cli",
	Short: "client for the the Qemu Storage Daemon",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&Port, "port", "p", "4444", "Port for the QMP server")
	rootCmd.PersistentFlags().StringVarP(&Host, "server", "s", "localhost", "Host for the QMP server")

}
