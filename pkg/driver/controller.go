package driver

import (
	"context"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateVolume creates a new volume from the given request. The function is
// idempotent.
func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Name must be provided")
	}

	if req.VolumeCapabilities == nil || len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Volume capabilities must be provided")
	}

	volumeName := req.Name
	size := req.GetCapacityRange()
	log := d.log.WithFields(logrus.Fields{
		"volume_name":         volumeName,
		"storage_size_":       size.RequiredBytes,
		"method":              "create_volume",
		"volume_capabilities": req.VolumeCapabilities,
	})
	if _, ok := d.storage[volumeName]; ok {
		log.Infof("volume %s already exists", volumeName)
	}
	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeName,
			CapacityBytes: size.RequiredBytes,
		},
	}
	log.WithField("response", resp).Info("volume was created")
	return resp, nil
}

// DeleteVolume deletes the given volume. The function is idempotent.
func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "DeleteVolume Volume ID must be provided")
	}

	log := d.log.WithFields(logrus.Fields{
		"volume_id": req.VolumeId,
		"method":    "delete_volume",
	})
	log.Info("delete volume called")

	d.deleteVolume(req.VolumeId)
	log.Info("volume was deleted")
	return &csi.DeleteVolumeResponse{}, nil
}
