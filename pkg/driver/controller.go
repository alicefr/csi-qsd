package driver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// createImageID cuts the ID it removes the pvc- prefix and takes the first 8 chars
func createImageID(ID string) string {
	return strings.TrimPrefix(ID, "pvc-")[:8]
}

// createSnapshotID cuts the ID it removes the snapshot- prefix and takes the first 8 chars
func createSnapshotID(ID string) string {
	return strings.TrimPrefix(ID, "snapshot-")[:8]
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
		// do not return an error because the volume might be already deleted
		log.Errorf("Failed to delete volume %s: because not found", req.VolumeId)
		return &csi.DeleteVolumeResponse{}, nil
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
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
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

func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	id := req.GetName()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "CreateSnapshot Name must be provided")
	}

	imageID := req.GetSourceVolumeId()
	if imageID == "" {
		return nil, status.Error(codes.InvalidArgument, "CreateSnapshot Source Volume ID must be provided")
	}

	log := d.log.WithFields(logrus.Fields{
		"req_name":             id,
		"req_source_volume_id": imageID,
		"req_parameters":       req.GetParameters(),
		"method":               "controller_create_snapshot",
	})

	// Retrieve base image
	source, ok := d.storage[imageID]
	if !ok {
		return nil, status.Errorf(codes.Internal, "Source volume %s not found in the storage", imageID)
	}
	if _, ok := d.snapshots[id]; ok {
		log.Info("Snapshot already created")
		return &csi.CreateSnapshotResponse{
			Snapshot: &csi.Snapshot{
				SnapshotId:     id,
				SourceVolumeId: imageID,
				ReadyToUse:     true,
			},
		}, nil

	}
	baseID := ""
	// It isn't the first snapshot
	if source.activeLayer != "" {
		baseID = source.activeLayer
	}
	s := Snapshot{
		baseID: baseID,
		node:   source.node,
		source: imageID,
	}

	log.Info("create snapshot is called")
	// Create client to the QSD grpc server on the node where the volume has to be created
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", s.node, d.port), opts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to connect to the QSD server for node %s:%v", source.node, err)
	}
	client := qsd.NewQsdServiceClient(conn)
	defer conn.Close()
	image := &qsd.Snapshot{
		ID:             createSnapshotID(id),
		SourceVolumeID: createImageID(imageID),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Create snapshot
	log.Info("create snapshot with the QSD")
	_, err = client.CreateSnapshot(ctx, image)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error in creating the snapshot %v", err)
	}
	// Snapshot successfully created store it
	d.snapshots[id] = s
	source.activeLayer = id
	log.Info("successfully add snapshot %v", s)
	tstamp, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return nil, fmt.Errorf("couldn't convert protobuf timestamp to go time.Time: %s",
			err.Error())
	}
	return &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			SnapshotId:     id,
			SourceVolumeId: baseID,
			ReadyToUse:     true,
			CreationTime:   tstamp,
		},
	}, nil
}

// DeleteSnapshot will be called by the CO to delete a snapshot.
func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	id := req.SnapshotId
	log := d.log.WithFields(logrus.Fields{
		"req_snapshot_id": req.GetSnapshotId(),
		"method":          "delete_snapshot",
	})

	s, ok := d.snapshots[id]
	if !ok {
		// do not return an error because the volume might be already deleted
		log.Errorf("Failed to delete volume %s: because not found", id)
		return &csi.DeleteSnapshotResponse{}, nil
	}
	log.Info("snapshot was deleted")
	// Create client to the QSD grpc server on the node where the volume has to be created
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", s.node, d.port), opts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to connect to the QSD server for node %s:%v", s.node, err)
	}
	client := qsd.NewQsdServiceClient(conn)
	defer conn.Close()
	image := &qsd.Snapshot{
		ID:             createSnapshotID(id),
		SourceVolumeID: createImageID(s.source),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// delete snapshot
	log.Info("delete snapshot with the QSD")
	_, err = client.DeleteSnapshot(ctx, image)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error for creating the exporter %v", err)
	}
	d.deleteSnapshot(id)
	return &csi.DeleteSnapshotResponse{}, nil
}
