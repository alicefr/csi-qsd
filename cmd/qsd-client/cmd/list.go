package cmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
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
		var r *qsd.ResponseListVolumes
		r, err = client.ListVolumes(ctx, &qsd.ListVolumesParams{})
		if err != nil {
			return fmt.Errorf("Error for creating the volume %v", err)
		}
		tree, err := cmd.Flags().GetBool("tree")
		if err != nil {
			log.Fatalf("Error getting tree flag: %v", err)
		}
		if tree {
			printTree(r)
			return nil
		}

		for _, v := range r.GetVolumes() {
			fmt.Println(v)
		}
		return nil
	},
}

type ByDepth []*qsd.Volume

func (a ByDepth) Len() int           { return len(a) }
func (a ByDepth) Less(i, j int) bool { return a[i].Depth < a[j].Depth }
func (a ByDepth) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func printTree(r *qsd.ResponseListVolumes) {
	var sortedImages []*qsd.Volume
	for _, v := range r.GetVolumes() {
		sortedImages = append(sortedImages, v)
	}
	sort.Sort(ByDepth(sortedImages))
	tree := treeprint.New()
	branches := make(map[string]treeprint.Tree)
	// Put the first nodes in the tree
	for _, v := range sortedImages {
		if v.Depth > 0 {
			break
		}
		branches[v.VolumeRef] = tree.AddMetaBranch(v.VolumeRef, v)
	}
	for _, v := range sortedImages {
		if v.Depth < 1 {
			continue
		}

		b, ok := branches[v.BackingImageID]
		if ok {
			branches[v.VolumeRef] = b.AddMetaBranch(v.VolumeRef, v)
		}
	}
	fmt.Println(tree.String())
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().Bool("tree", false, "Print tree")
}
