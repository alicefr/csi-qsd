package driver

import (
	"context"
	"fmt"
	"os"
	"syscall"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	log := s.log.WithFields(logrus.Fields{
		"volume_id": req.VolumeId,
		"method":    "node_publish_volume",
	})

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
	//isFsVol := req.GetVolumeCapability().GetMount() != nil
	if isBlockVol {
		return nil, status.Error(codes.InvalidArgument, "Block devices not supported")
	}
	// Create target directory
	err := os.MkdirAll(req.GetTargetPath(), 0755)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mkdir failed: target=%s, error=%v", req.GetTargetPath(), err)
	}

	// Mount vhost-user socket dir into the target directory
	socketDir := fmt.Sprintf("%s/%s", SocketDir, volumeID)
	//socket := fmt.Sprintf("%s/%s/%s", socketDir, vhostSock)
	//if _, err := os.Stat(socket); err != nil {
	//	return nil, status.Errorf(codes.Internal, "failed in stating the socket for volume %s: %v", volumeID, err)
	//}

	if err := syscall.Mount(socketDir, req.GetTargetPath(), "none", syscall.MS_BIND, ""); err != nil {
		return nil, status.Errorf(codes.Internal, "failed in mounting the socket dir for volume %s: %v", volumeID, err)
	}

	log.Info("volume published on the node")

	return &csi.NodePublishVolumeResponse{}, nil

}

func (s *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	log := s.log.WithFields(logrus.Fields{
		"volume_id": req.VolumeId,
		"method":    "node_unpublish_volume",
	})
	volumeID := req.GetVolumeId()
	err := syscall.Unmount(req.GetTargetPath(), 0)
	if err != nil && !os.IsNotExist(err) {
		return nil, status.Errorf(codes.Internal, "failed in unmounting the target dir for volume %s: %v", volumeID, err)
	}
	if err := os.RemoveAll(req.GetTargetPath()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed in removing the target dir for volume %s: %v", volumeID, err)
	}
	log.Info("volume unpublished")
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (s *Driver) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
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

func (s *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: s.nodeId,
	}, nil
}
