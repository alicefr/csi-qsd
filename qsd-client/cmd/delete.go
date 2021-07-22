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

// deleteCmd represents the create command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a image",
	RunE: func(cmd *cobra.Command, args []string) error {
		image, err := cmd.Flags().GetString("image")
		if err != nil {
			log.Fatalf("Error getting image exporter: %v", err)
		}
		i := &qsd.Image{
			ID: image,
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// Remove exporter
		log.Info("remove exporter with the QSD")
		_, err = client.DeleteExporter(ctx, i)
		if err != nil {
			return fmt.Errorf("Error for creating the exporter %v", err)
		}
		// Remove Volume
		log.Info("remove backend image with the QSD")
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err = client.DeleteVolume(ctx, i)
		if err != nil {
			return fmt.Errorf("Error for creating the volume %v", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().String("image", "image", "Name of the image")
	deleteCmd.MarkFlagRequired("image")
}
