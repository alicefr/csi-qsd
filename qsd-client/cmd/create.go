package cmd

import (
	"github.com/alicefr/csi-qsd/pkg/qsd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"strconv"
)

func createVolumeUnixSocket(socket, image string, size int) {
	m, err := qsd.CreateNewUnixMonitor(socket)
	if err != nil {
		log.Fatalf("Error creating monitor: %v", err)
	}
	v := &qsd.Volume{Monitor: m}
	defer v.Monitor.Disconnect()
	log.Infof("Create Volume")
	err = v.CreateVolume(image)
	if err != nil {
		log.Fatalf("Error creating volume: %v", err)
	}

}

func createVolumeTCPSocket(host, port, image string, size int) {
	m, err := qsd.CreateNewTCPMonitor(host, port)
	if err != nil {
		log.Fatalf("Error creating monitor: %v", err)
	}
	v := &qsd.Volume{Monitor: m, Size: strconv.Itoa(size)}
	defer v.Monitor.Disconnect()
	log.Infof("Create Volume")
	err = v.CreateVolume(image)
	if err != nil {
		log.Fatalf("Error creating volume: %v", err)
	}

}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a image",
	Long:  `Create a image`,
	Run: func(cmd *cobra.Command, args []string) {
		image, err := cmd.Flags().GetString("image")
		if err != nil {
			log.Fatalf("Error getting image exporter: %v", err)
		}
		var size int
		size, err = cmd.Flags().GetInt("size")
		if err != nil {
			log.Fatalf("Error getting size exporter: %v", err)
		}
		switch {
		case Socket != "":
			createVolumeUnixSocket(Socket, image, size)
		case Host != "" && Port != "":
			createVolumeTCPSocket(Host, Port, image, size)
		default:
			log.Fatalf("You need to specify either unix or tcp socket")
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	createCmd.Flags().String("image", "image", "Name of the image")
	createCmd.Flags().Int("size", 0, "Size of the image")
	createCmd.MarkFlagRequired("image")
	createCmd.MarkFlagRequired("size")
}
