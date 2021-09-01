package qsd

import (
	context "context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	imagesDir      = "/var/run/qsd/images"
	socketDir      = "/var/run/qsd/sockets"
	diskImg        = "disk.img"
	vhostSock      = "vhost.sock"
	snapshotPrefix = "snap"
)

type QCOWImage struct {
	QSDID          string
	BackingImageID string
	File           string
	RefCount       int32
	VolumeRef      string
	Depth          uint32
}

type Server struct {
	QsdServiceServer
	qsdSock      string
	images       map[string]*QCOWImage
	activeLayers map[string]string
	volManager   *VolumeManager
}

func NewServer(sock string) (*Server, error) {
	volManager, err := NewVolumeManager(sock)
	if err != nil {
		return nil, fmt.Errorf("Failed creating the qsd monitor connection")
	}
	return &Server{
		qsdSock:      sock,
		images:       make(map[string]*QCOWImage),
		activeLayers: make(map[string]string),
		volManager:   volManager,
	}, nil
}

func (c *Server) Disconnect() {
	c.volManager.Disconnect()
}

func failed(m string, err error) (*Response, error) {
	log.Errorf(m)
	return &Response{
		Success: false,
		Message: m,
	}, err

}

func generateQSDID(id string) string {
	size := 8
	if len(id) < size {
		size = len(id)
	}
	if strings.Contains(id, "pvc-") {
		return strings.TrimPrefix(id, "pvc-")[:size]
	}

	if strings.Contains(id, "snapshot-") {
		return strings.TrimPrefix(id, "snapshot-")[:size]
	}
	return id[:size]
}

func (c *Server) CreateVolume(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Create Volume %s", image.ID)
	dir := fmt.Sprintf("%s/%s", imagesDir, image.ID)
	// Create directory for the volume if it doesn't exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		errMessage := fmt.Sprintf("Cannot create directory for the volume:%s", image.ID)
		return failed(errMessage, err)
	}
	qcowImage := &QCOWImage{
		File:      fmt.Sprintf("%s/%s", dir, diskImg),
		QSDID:     generateQSDID(image.ID),
		RefCount:  0,
		VolumeRef: image.ID,
	}
	if image.FromVolume == "" {
		if err := c.volManager.CreateVolume(qcowImage.File, qcowImage.QSDID, strconv.FormatInt(image.Size, 10)); err != nil {
			errMessage := fmt.Sprintf("Failed creating the disk image %s:%v", image.ID, err)
			return failed(errMessage, err)
		}
		qcowImage.Depth = 0
	} else {
		log.Infof("Create image %s from %s", image.ID, image.FromVolume)
		qcowImage.BackingImageID = image.FromVolume
		b, ok := c.images[image.FromVolume]
		if !ok {
			return &Response{}, fmt.Errorf("Failed to delete the image %s: image not found", image.FromVolume)
		}
		if err := c.volManager.CreateSnapshotWithBackingNode(b.QSDID, qcowImage.QSDID, b.File, qcowImage.File, b.QSDID); err != nil {
			errMessage := fmt.Sprintf("Cannot snapshot %s: %v", image.FromVolume, err)
			return failed(errMessage, err)
		}
		b.RefCount++
		c.images[image.FromVolume] = b
		qcowImage.Depth = b.Depth + 1

	}
	c.images[image.ID] = qcowImage
	c.activeLayers[image.ID] = image.ID
	return &Response{
		Success: true,
	}, nil
}

func (c *Server) ExposeVhostUser(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Export vhost user for image %s", image.ID)
	i, ok := c.images[image.ID]
	if !ok {
		errMessage := fmt.Sprintf("Image %s not found", image.ID)
		return failed(errMessage, fmt.Errorf(errMessage))
	}
	dir := fmt.Sprintf("%s/%s", socketDir, image.ID)
	// Create directory for the socket if it doesn't exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		errMessage := fmt.Sprintf("Cannot create socket directory for the volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	socket := fmt.Sprintf("%s/%s", dir, vhostSock)
	// Expose and create vhost-user socket
	if _, err := os.Stat(socket); os.IsExist(err) {
		if err := os.Remove(socket); err != nil {
			errMessage := fmt.Sprintf("Cannot create socket directory for the volume %s: %v", image.ID, err)
			return failed(errMessage, err)
		}
	}

	if err := c.volManager.ExposeVhostUser(i.QSDID, socket); err != nil {
		errMessage := fmt.Sprintf("Cannot create socket for the volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	return &Response{
		Success: true,
	}, nil

}

func deleteIfEmptyDir(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	if len(files) != 0 {
		return nil
	}

	return os.Remove(path)
}

func (c *Server) DeleteExporter(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Delete exporter %s", image.ID)
	i, ok := c.images[image.ID]
	if !ok {
		errMessage := fmt.Sprintf("Image %s not found", image.ID)
		return failed(errMessage, fmt.Errorf(errMessage))
	}
	if err := c.volManager.DeleteExporter(i.QSDID); err != nil {
		errMessage := fmt.Sprintf("Cannot delete exporter for volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	dir := fmt.Sprintf("%s/%s", socketDir, image.ID)
	if err := os.Remove(dir); err != nil {
		errMessage := fmt.Sprintf("Cannot delete socket directory for volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	return &Response{}, nil
}

func (c *Server) CreateSnapshot(ctx context.Context, snapshot *Snapshot) (*Response, error) {
	log.Infof("Create Snapshot %s of image %s", snapshot.ID, snapshot.SourceVolumeID)
	// Get active layer of the image
	id, ok := c.activeLayers[snapshot.SourceVolumeID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: active layer not found", snapshot.SourceVolumeID)
	}
	var i *QCOWImage
	i, ok = c.images[id]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: image not found", snapshot.SourceVolumeID)
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, snapshot.SourceVolumeID)
	if _, err := os.Stat(dir); err != nil {
		errMessage := fmt.Sprintf("Failed checking the directory for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	s := &QCOWImage{
		QSDID:          generateQSDID(snapshot.ID),
		BackingImageID: id,
		File:           fmt.Sprintf("%s/%s-%s", dir, snapshotPrefix, generateQSDID(snapshot.ID)),
		VolumeRef:      snapshot.ID,
		Depth:          i.Depth + 1,
	}
	if i.RefCount < 1 {
		if err := c.volManager.CreateSnapshot(i.QSDID, s.QSDID, i.File, s.File); err != nil {
			errMessage := fmt.Sprintf("Cannot snapshot %s: %v", snapshot.ID, err)
			return failed(errMessage, err)
		}
	} else {
		if err := c.volManager.CreateSnapshotWithBackingNode(i.QSDID, s.QSDID, i.File, s.File, i.QSDID); err != nil {
			errMessage := fmt.Sprintf("Cannot snapshot %s: %v", snapshot.ID, err)
			return failed(errMessage, err)
		}

	}
	i.RefCount++
	c.images[id] = i
	c.images[snapshot.ID] = s
	// Update the active layer with the new snapshot
	c.activeLayers[snapshot.SourceVolumeID] = snapshot.ID
	return &Response{}, nil

}

func (c *Server) DeleteVolume(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Delete image %s", image.ID)
	// Get the active layer of the image
	id, ok := c.activeLayers[image.ID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: active layer not found", image.ID)
	}
	var i *QCOWImage
	i, ok = c.images[id]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: image not found", image.ID)
	}
	if i.RefCount < 1 && id == image.ID {
		if err := c.deleteImage(id); err != nil {
			errMessage := fmt.Sprintf("Failed deleting image %s:%v", image.ID, err)
			return failed(errMessage, err)
		}
	} else {
		// If the active layer is a snapshot then remove the count reference for the image
		i.RefCount--
		c.images[id] = i
		// Remove the volume reference from the image
		i, ok = c.images[image.ID]
		if ok {
			i.VolumeRef = ""
			c.images[image.ID] = i
		}
	}
	delete(c.activeLayers, image.ID)
	dir := fmt.Sprintf("%s/%s", imagesDir, image.ID)
	if err := deleteIfEmptyDir(dir); err != nil {
		errMessage := fmt.Sprintf("Cannot delete image directory for the volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	if err := c.deleteNodeWithZeroReference(i.BackingImageID); err != nil {
		errMessage := fmt.Sprintf("Failed cleaning up the zero reference node %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	return &Response{}, nil
}

func (c *Server) deleteNodeWithZeroReference(id string) error {
	log.Infof("Cleaning node with zero references %s", id)
	i, ok := c.images[id]
	// The image has already been deleted
	if !ok {
		return nil
	}
	// Don't delete node if there is still a node pointing to the image or a volume reference
	if i.RefCount > 0 || i.VolumeRef != "" {
		return nil
	}

	if err := c.deleteImage(id); err != nil {
		return err
	}

	if i.BackingImageID != "" {
		return c.deleteNodeWithZeroReference(i.BackingImageID)
	}
	return nil
}

func (c *Server) deleteImage(id string) error {
	i, ok := c.images[id]
	if !ok {
		return fmt.Errorf("Image %s not found", id)
	}
	if err := c.volManager.DeleteVolume(i.QSDID); err != nil {
		return err
	}
	if err := os.Remove(i.File); err != nil {
		return err
	}
	if i.BackingImageID != "" {
		// Decrease count reference of the backing image
		b, ok := c.images[i.BackingImageID]
		if !ok {
			return fmt.Errorf("Backing image %s not found", i.BackingImageID)
		}

		b.RefCount--
		c.images[i.BackingImageID] = b
	}
	delete(c.images, id)
	return nil

}

func (c *Server) DeleteSnapshot(ctx context.Context, snapshot *Snapshot) (*Response, error) {
	log.Infof("Delete snapshot %s", snapshot.ID)
	s, ok := c.images[snapshot.ID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to get snapshot to delete %s: image not found", snapshot.SourceVolumeID)
	}
	// Get active layer of the image
	id, ok := c.activeLayers[snapshot.SourceVolumeID]
	isActiveLayer := ok && id == snapshot.ID

	if s.RefCount < 1 && !isActiveLayer {
		if err := c.deleteImage(snapshot.ID); err != nil {
			errMessage := fmt.Sprintf("Failed deleting snapshot %s:%v", snapshot.ID, err)
			return failed(errMessage, err)

		}
	} else {
		s.VolumeRef = ""
		c.images[snapshot.ID] = s
	}
	if err := c.deleteNodeWithZeroReference(s.BackingImageID); err != nil {
		errMessage := fmt.Sprintf("Failed cleaning up the zero reference node %s: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	return &Response{}, nil
}

func (c *Server) ListVolumes(ctx context.Context, _ *ListVolumesParams) (*ResponseListVolumes, error) {
	log.Infof("List the images")
	var volumes []*Volume
	for k, v := range c.images {
		volumes = append(volumes, &Volume{
			QSDID:          v.QSDID,
			BackingImageID: v.BackingImageID,
			File:           v.File,
			RefCount:       v.RefCount,
			Depth:          v.Depth,
			VolumeRef:      k,
		})
	}
	return &ResponseListVolumes{
		Volumes: volumes,
	}, nil
}
