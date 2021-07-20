package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("driver").WithName("controller")

// CreateVolume creates a new volume from the given request. The function is
// idempotent.
func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Name must be provided")
	}

	volumeName := req.Name
	size := req.GetCapacityRange()
	log.Info("CreateVolume called",
		"volume_name", volumeName,
		"storage_size_", size.RequiredBytes,
		"method", "create_volume",
		"volume_capabilities", req.VolumeCapabilities,
	)
	if _, ok := d.storage[volumeName]; !ok {
		d.storage[volumeName] = size.RequiredBytes
	}
	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeName,
			CapacityBytes: size.RequiredBytes,
		},
	}
	log.Info("response", "volume was created")
	return resp, nil
}

// DeleteVolume deletes the given volume. The function is idempotent.
func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "DeleteVolume Volume ID must be provided")
	}

	log.Info("DeleteVolume called",
		"volume_id", req.VolumeId,
		"method", "delete_volume",
	)

	d.deleteVolume(req.VolumeId)
	log.Info("volume was deleted")
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	log.Info("ControllerPublishVolume called",
		"volume_id", req.VolumeId,
		"node_id", req.NodeId,
		"method", "publish_volume",
	)
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID must be provided")
	}

	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeIP must be provided")

	}
	size, ok := d.storage[req.VolumeId]
	if !ok {
		return nil, status.Errorf(codes.Internal, "Volume ID not found %s", req.VolumeId)
	}
	image := &qsd.Image{
		ID:   req.VolumeId,
		Size: size,
	}
	// Create client to the QSD grpc server on the node where the volume has to be created
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", req.NodeId, d.port), opts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to connect to the QSD server for node %s:%v", req.NodeId, err)
	}
	client := qsd.NewQsdServiceClient(conn)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create Volume
	var r *qsd.Response
	r, err = client.CreateVolume(ctx, image)
	if !r.Success {
		return nil, status.Error(codes.Internal, r.Message)
	}
	// Create exporter
	r, err = client.ExposeVhostUser(ctx, image)
	if !r.Success {
		return nil, status.Error(codes.Internal, r.Message)
	}

	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	log.Info("ControllerPublishVolume called",
		"volume_id", req.VolumeId,
		"method", "unpublish_volume",
	)
	// TODO delete exporter from the QSD and remove the qcow image
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}
