package driver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// createImageID cuts the ID it removes the pvc- prefix and takes the first 8 chars
func createImageID(ID string) string {
	return strings.TrimPrefix(ID, "pvc-")[:8]
}

// CreateVolume creates a new volume from the given request. The function is
// idempotent.
func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Name must be provided")
	}

	volumeName := req.Name
	size := req.GetCapacityRange()
	log := d.log.WithFields(logrus.Fields{
		"volume_id": req.Name,
		"size":      size.RequiredBytes,
		"method":    "controller_create_volume",
	})
	id := createImageID(req.Name)
	// TODO smart scheduling we need already to know where the volume has to be created
	// HACK hard code the node just for PoC
	v, ok := d.storage[volumeName]
	if !ok {
		v = Volume{
			id:   id,
			size: size.RequiredBytes,
			node: "k8s-qsd-control-plane",
		}
		d.storage[volumeName] = v
	}

	image := &qsd.Image{
		ID:   id,
		Size: v.size,
	}
	// Create client to the QSD grpc server on the node where the volume has to be created
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", v.node, d.port), opts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to connect to the QSD server for node %s:%v", v.node, err)
	}
	client := qsd.NewQsdServiceClient(conn)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Create Volume
	log.Info("create backend image with the QSD")
	var r *qsd.Response
	r, err = client.CreateVolume(ctx, image)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error for creating the volume %v", err)
	}
	if !r.Success {
		return nil, status.Error(codes.Internal, r.Message)
	}
	// Create exporter
	log.Info("create exporter with the QSD")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err = client.ExposeVhostUser(ctx, image)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error for creating the exporter %v", err)
	}
	if !r.Success {
		return nil, status.Error(codes.Internal, r.Message)
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
	log := d.log.WithFields(logrus.Fields{
		"volume_id": req.VolumeId,
		"method":    "controller_delete_volume",
	})
	v, ok := d.storage[req.VolumeId]
	if !ok {
		return nil, status.Errorf(codes.Internal, "Failed to delete volume %s: because not found", req.VolumeId)
	}
	// Create client to the QSD grpc server on the node where the volume has to be created
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", v.node, d.port), opts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to connect to the QSD server for node %s:%v", v.node, err)
	}
	client := qsd.NewQsdServiceClient(conn)
	defer conn.Close()
	image := &qsd.Image{
		ID: v.id,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Remove exporter
	log.Info("remove exporter with the QSD")
	_, err = client.DeleteExporter(ctx, image)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error for creating the exporter %v", err)
	}
	// Remove Volume
	log.Info("remove backend image with the QSD")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = client.DeleteVolume(ctx, image)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error for creating the volume %v", err)
	}

	d.deleteVolume(req.VolumeId)
	log.Info("volume was deleted")
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerGetCapabilities(context.Context, *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	capabilities := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	}

	csiCaps := make([]*csi.ControllerServiceCapability, len(capabilities))
	for i, capability := range capabilities {
		csiCaps[i] = &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: capability,
				},
			},
		}
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: csiCaps,
	}, nil
}
