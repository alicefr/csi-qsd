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
	RefCount       int
}

type Server struct {
	QsdServiceServer
	qsdSock      string
	images       map[string]QCOWImage
	activeLayers map[string]string
}

func NewServer(sock string) *Server {
	return &Server{
		qsdSock:      sock,
		images:       make(map[string]QCOWImage),
		activeLayers: make(map[string]string),
	}
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
	log.Infof("Create new monitor for the volume creation")
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err)
		return failed(errMessage, err)
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, image.ID)
	i := fmt.Sprintf("%s/%s", dir, diskImg)
	// Create directory for the volume if it doesn't exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		errMessage := fmt.Sprintf("Cannot create directory for the volume:%s", image.ID)
		return failed(errMessage, err)
	}
	qsdID := generateQSDID(image.ID)
	if err := volManager.CreateVolume(i, qsdID, strconv.FormatInt(image.Size, 10)); err != nil {
		errMessage := fmt.Sprintf("Failed creating the disk image %s:%v", image.ID, err)
		return failed(errMessage, err)
	}

	_, err = os.Stat(i)
	if err != nil {
		errMessage := fmt.Sprintf("Failed stating the image %s:%v", image.ID, err)
		return failed(errMessage, err)
	}
	c.images[image.ID] = QCOWImage{
		File:     i,
		QSDID:    qsdID,
		RefCount: 0,
	}
	c.activeLayers[image.ID] = image.ID
	return &Response{
		Success: true,
	}, nil
}

func (c *Server) ExposeVhostUser(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Create new monitor to expose vhost user")
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err)
		return failed(errMessage, err)
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

	if err := volManager.ExposeVhostUser(image.ID, socket); err != nil {
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

func (c *Server) DeleteVolume(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Create new monitor to delete volume")
	// Get the active layer of the image
	id, ok := c.activeLayers[image.ID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: active layer not found", image.ID)
	}
	var i QCOWImage
	i, ok = c.images[id]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: image not found", image.ID)
	}
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err)
		return failed(errMessage, err)
	}
	if err := volManager.DeleteVolume(i.QSDID); err != nil {
		errMessage := fmt.Sprintf("Cannot delete volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}

	delete(c.activeLayers, image.ID)
	// If there are no snapshot we can remove file
	if i.RefCount == 0 {
		if err := os.Remove(i.File); err != nil {
			errMessage := fmt.Sprintf("Cannot delete image for the volume %s: %v", image.ID, err)
			return failed(errMessage, err)
		}
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, image.ID)
	if err := deleteIfEmptyDir(dir); err != nil {
		errMessage := fmt.Sprintf("Cannot delete image directory for the volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	return &Response{}, nil
}

func (c *Server) DeleteExporter(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Create new monitor to delete exporter")
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err)
		return failed(errMessage, err)
	}
	if err := volManager.DeleteExporter(image.ID); err != nil {
		errMessage := fmt.Sprintf("Cannot delete exporter for volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	// The socket directory will be unmounted and deleted by the driver
	return &Response{}, nil
}

func (c *Server) CreateSnapshot(ctx context.Context, snapshot *Snapshot) (*Response, error) {
	log.Infof("Create new monitor to snapshot")
	// Get active layer of the image
	id, ok := c.activeLayers[snapshot.SourceVolumeID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: active layer not found", snapshot.SourceVolumeID)
	}
	var i QCOWImage
	i, ok = c.images[id]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: image not found", snapshot.SourceVolumeID)
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, snapshot.SourceVolumeID)
	if _, err := os.Stat(dir); err != nil {
		errMessage := fmt.Sprintf("Failed checking the directory for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	s := QCOWImage{
		QSDID:          generateQSDID(snapshot.ID),
		BackingImageID: id,
		File:           fmt.Sprintf("%s/%s-%s", dir, snapshotPrefix, generateQSDID(snapshot.ID)),
	}
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	if err := volManager.CreateSnapshot(i.QSDID, s.QSDID, i.File, s.File); err != nil {
		errMessage := fmt.Sprintf("Cannot snapshot %s: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	c.images[snapshot.ID] = s
	// Update the active layer with the new snapshot
	c.activeLayers[snapshot.SourceVolumeID] = snapshot.ID
	return &Response{}, nil

}

func (c *Server) DeleteSnapshot(ctx context.Context, snapshot *Snapshot) (*Response, error) {
	log.Infof("Create new monitor to delete snapshot")
	s, ok := c.images[snapshot.ID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to get snapshot to delete %s: image not found", snapshot.SourceVolumeID)
	}
	var b QCOWImage
	b, ok = c.images[s.BackingImageID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to get backing image %s: image not found", s.BackingImageID)
	}
	//Get active layer of the image
	id, ok := c.activeLayers[snapshot.SourceVolumeID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: active layer not found", snapshot.SourceVolumeID)
	}
	var i QCOWImage
	i, ok = c.images[id]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete snapshot %s: image not found", snapshot.SourceVolumeID)
	}

	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	if err := volManager.CommitImage(i.QSDID, s.File, b.File); err != nil {
		errMessage := fmt.Sprintf("Cannot copy snapshot %s in the upper layer: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}

	if err := volManager.DeleteVolume(s.QSDID); err != nil {
		errMessage := fmt.Sprintf("Cannot node for snapshot %s: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}

	if err := os.Remove(s.File); err != nil {
		errMessage := fmt.Sprintf("Cannot delete snapshot %s: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, snapshot.SourceVolumeID)
	if err := deleteIfEmptyDir(dir); err != nil {
		errMessage := fmt.Sprintf("Cannot delete image directory for the volume %s: %v", snapshot.SourceVolumeID, err)
		return failed(errMessage, err)
	}

	return &Response{}, nil
}

func (c *Server) ListVolumes(ctx context.Context, _ *ListVolumesParams) (*Response, error) {
	log.Infof("Create new monitor to list the volumes")
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor:%s:%v", err)
		return failed(errMessage, err)
	}
	nodes, err := volManager.GetNameBlockNodes()
	if err != nil {
		errMessage := fmt.Sprintf("Cannot list volumes: %v", err)
		return failed(errMessage, err)
	}
	return &Response{
		Success: true,
		Message: fmt.Sprintf("Volumes: %v", nodes),
	}, nil
}
