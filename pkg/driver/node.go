package driver

import (
	"context"
	"fmt"
	"os"
	"syscall"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	ctrl "sigs.k8s.io/controller-runtime"
)

var nodeLogger = ctrl.Log.WithName("driver").WithName("node")

func (s *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
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
	//isBlockVol := req.GetVolumeCapability().GetBlock() != nil
	isFsVol := req.GetVolumeCapability().GetMount() != nil
	if isFsVol {
		return nil, status.Error(codes.InvalidArgument, "FUSE not supported yet")
	}
	// Create target directory
	err := os.MkdirAll(req.GetTargetPath(), 0755)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mkdir failed: target=%s, error=%v", req.GetTargetPath(), err)
	}
	// Mount vhost-user socket dir into the target directory
	socketDir := fmt.Sprintf("%s/%s", SocketDir, volumeID)
	socket := fmt.Sprintf("%s/%s/%s", socketDir, vhostSock)
	if _, err := os.Stat(socket); err != nil {
		return nil, status.Errorf(codes.Internal, "failed in stating the socket for volume %s: %v", volumeID, err)
	}

	if err := syscall.Mount(socketDir, req.GetTargetPath(), "none", syscall.MS_BIND, ""); err != nil {
		return nil, status.Errorf(codes.Internal, "failed in mounting the socket dir for volume %s: %v", volumeID, err)
	}

	nodeLogger.Info("node publish volume called")

	return &csi.NodePublishVolumeResponse{}, nil

}

func (s *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	// Unmount and remove the socket directory
	socketDir := fmt.Sprintf("%s/%s", SocketDir, volumeID)
	if err := syscall.Unmount(socketDir, 0); err != nil {
		return nil, status.Errorf(codes.Internal, "failed in unmounting the socket dir for volume %s: %v", volumeID, err)
	}
	if err := os.RemoveAll(socketDir); err != nil {
		return nil, status.Errorf(codes.Internal, "failed in removing the socket dir for volume %s: %v", volumeID, err)
	}
	if err := os.RemoveAll(req.GetTargetPath()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed in removing the socket dir for volume %s: %v", volumeID, err)
	}
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
