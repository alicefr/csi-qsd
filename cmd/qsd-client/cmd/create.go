package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a image",
	Long:  `Create a image`,
	RunE: func(cmd *cobra.Command, args []string) error {
		image, err := cmd.Flags().GetString("image")
		if err != nil {
			log.Fatalf("Error getting image exporter: %v", err)
		}
		source, err := cmd.Flags().GetString("from")
		if err != nil {
			log.Fatalf("Error getting source of the image: %v", err)
		}

		var size int64
		size, err = cmd.Flags().GetInt64("size")
		if err != nil && source == "" {
			log.Fatalf("Error getting size exporter: %v", err)
		}
		i := &qsd.Image{
			ID:         image,
			Size:       size,
			FromVolume: source,
		}
		// Create client to the QSD grpc server on the node where the volume has to be created
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(fmt.Sprintf("%s:%s", Host, Port), opts...)
		if err != nil {
			return fmt.Errorf("Failed to connect to the QSD server:%v", err)
		}
		client := qsd.NewQsdServiceClient(conn)
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		// Create Volume
		log.Info("create backend image with the QSD")
		_, err = client.CreateVolume(ctx, i)
		if err != nil {
			return fmt.Errorf("Error for creating the volume %v", err)
		}
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		log.Info("create exporter with the QSD")
		_, err = client.ExposeVhostUser(ctx, i)
		if err != nil {
			return fmt.Errorf("Error for creating the exporter %v", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().String("image", "image", "Name of the image")
	createCmd.Flags().Int64("size", 0, "Size of the image")
	createCmd.Flags().String("from", "", "Name of the image to use as source to create the snapshot")
	createCmd.MarkFlagRequired("image")
}
