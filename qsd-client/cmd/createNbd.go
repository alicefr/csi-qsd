package cmd

import (
	"github.com/alicefr/csi-qsd/pkg/qsd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func createNbd(socket, path, exporter string) {
	m, err := qsd.CreateNewUnixMonitor(socket)
	if err != nil {
		log.Fatalf("Error creating monitor: %v", err)
	}
	v := &qsd.Volume{Monitor: m}
	defer v.Monitor.Disconnect()
	log.Infof("Create Nbd server")
	err = v.CreateNbdServer(exporter, path)
	if err != nil {
		log.Fatalf("Error create ndb server: %v", err)
	}

}

var createNbdCmd = &cobra.Command{
	Use:   "create-nbd",
	Short: "Create ndb server",
	Run: func(cmd *cobra.Command, args []string) {
		var exporter string
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			log.Fatalf("Error getting flag path: %v", err)
		}
		exporter, err = cmd.Flags().GetString("exporter")
		if err != nil {
			log.Fatalf("Error getting flag exporter: %v", err)
		}
		createNbd(Socket, path, exporter)
	},
}

func init() {
	rootCmd.AddCommand(createNbdCmd)

	createNbdCmd.Flags().String("path", "", "Path for the unix nbd.sock")
	createNbdCmd.Flags().String("exporter", "", "Name of the nbd exporter")
	createNbdCmd.MarkFlagRequired("path")
	createNbdCmd.MarkFlagRequired("exporter")
}
