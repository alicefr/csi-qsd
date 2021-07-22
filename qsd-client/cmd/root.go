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
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&Port, "port", "p", "", "Port for the QMP server")
	rootCmd.PersistentFlags().StringVarP(&Host, "server", "q", "", "Host for the QMP server")

}
