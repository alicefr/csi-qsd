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

// listCmd represents the create command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List images",
	RunE: func(cmd *cobra.Command, args []string) error {
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
		// ListVolumes
		log.Info("create backend image with the QSD")
		var r *qsd.Response
		r, err = client.ListVolumes(ctx, &qsd.ListVolumesParams{})
		if err != nil {
			return fmt.Errorf("Error for creating the volume %v", err)
		}
		log.Info(r.Message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
