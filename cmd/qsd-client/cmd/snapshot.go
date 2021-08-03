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

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Snapshot command",
}

var snapshotCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		image, err := cmd.Flags().GetString("image")
		if err != nil {
			log.Fatalf("Error getting image exporter: %v", err)
		}
		var name string
		name, err = cmd.Flags().GetString("name")
		if err != nil {
			log.Fatalf("Error getting name for the snapshot: %v", err)
		}
		var source string
		source, err = cmd.Flags().GetString("source")
		if err != nil {
			log.Fatalf("Error getting the source for the snapshot: %v", err)
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
		s := &qsd.Snapshot{
			ID:               name,
			SourceVolumeID:   source,
			VolumeToSnapshot: image,
		}
		// Create Snapshot
		log.Info("create snapshot with the QSD")
		_, err = client.CreateSnapshot(ctx, s)
		if err != nil {
			return fmt.Errorf("Error for creating the volume %v", err)
		}

		return nil
	},
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var name string
		name, err = cmd.Flags().GetString("name")
		if err != nil {
			log.Fatalf("Error getting name for the snapshot: %v", err)
		}
		var source string
		source, err = cmd.Flags().GetString("source")
		if err != nil {
			log.Fatalf("Error getting the source for the snapshot: %v", err)
		}
		top, err := cmd.Flags().GetString("top-layer")
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
		s := &qsd.Snapshot{
			ID:             name,
			SourceVolumeID: source,
			UpperLayer:     top,
		}
		// Delete Snapshot
		log.Info("delete snapshot with the QSD")
		_, err = client.DeleteSnapshot(ctx, s)
		if err != nil {
			return fmt.Errorf("Error for creating the volume %v", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
	snapshotCmd.PersistentFlags().String("image", "", "Name of the image to snapshot")
	snapshotCmd.PersistentFlags().String("source", "", "Source of the snapshot")
	snapshotCmd.PersistentFlags().String("name", "", "Name of the snapshot")
	snapshotCmd.MarkFlagRequired("image")
	snapshotCmd.MarkFlagRequired("name")
	snapshotCmd.MarkFlagRequired("source")
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	snapshotDeleteCmd.PersistentFlags().String("top-layer", "", "Name of the upperlayer")
}
