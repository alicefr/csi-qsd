package driver

import (
	"context"
	"fmt"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ImageDir = "/images"
)

type nodeService struct {
	csi.UnimplementedNodeServer
	volManager *qsd.VolumeManager
	nodeName   string
}

var nodeLogger = ctrl.Log.WithName("driver").WithName("node")

// NewNodeService returns a new NodeServer.
func NewNodeService(nodeName string, m *qsd.VolumeManager) csi.NodeServer {
	return &nodeService{
		nodeName:   nodeName,
		volManager: m,
	}
}

func (s *nodeService) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	nodeLogger.Info("NodePublishVolume called",
		"volume_id", volumeID,
		"target_path", req.GetTargetPath(),
		"volume_capability", req.GetVolumeCapability(),
		"read_only", req.GetReadonly())

	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no volume_id is provided")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no target_path is provided")
	}
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "no volume_capability is provided")
	}
	isBlockVol := req.GetVolumeCapability().GetBlock() != nil
	isFsVol := req.GetVolumeCapability().GetMount() != nil
	if isFsVol {
		return error, status.Error(codes.InvalidArgument, "FUSE not supported yet")
	}

	// Create directory for the volume if it doesn't exists
	dir := fmt.Sprintf("%s/%s", ImageDir, volumeID)
	if err := os.MkDirAll(dir, 0755); err != nil {
		return status.Error(codes.Internal, "Cannot create directory for the volume:%s", volumeID)
	}

	// Create qcow2 image
	image := fmt.Sprintf("%s/%s", dir, "disk.img")
	if _, err := os.Stat(image); os.IsNotExist(err) {
		if err := volManager.CreateVolume(image, volumeID); err != nil {
			return status.Error(codes.Internal, "Failed in creating the disk image %s:%v", volumeID, err)
		}
	}

	// Create target directory
	err = os.MkdirAll(req.GetTargetPath(), 0755)
	if err != nil {
		return status.Errorf(codes.Internal, "mkdir failed: target=%s, error=%v", req.GetTargetPath(), err)
	}

	// Expose and create vhost-user socket
	if _, err := os.Stat(image); os.IsExist(err) {
		if err := os.Remove(image); err != nil {
			return status.Error(codes.Internal, "Failed in creating the disk image %s:%v", volumeID, err)
		}
	}

	if err := volManager.ExposeVhostUser(volumeID, image); err != nil {
		return status.Error(codes.Internal, "Failed exporting the disk image %s:%v", volumeID, err)
	}

	return &csi.NodePublishVolumeResponse{}, nil

}

func (s *nodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	// Delete the exporter

	// Delete the qcow image

	// Remove the directory
}

func (s *nodeService) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	capabilities := []csi.NodeServiceCapability_RPC_Type{}

	csiCaps := make([]*csi.NodeServiceCapability, len(capabilities))
	for i, capability := range capabilities {
		csiCaps[i] = &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: capability,
				},
			},
		}
	}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: csiCaps,
	}, nil
}

func (s *nodeService) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: s.nodeName,
	}, nil
}
