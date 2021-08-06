package qsd

import (
	context "context"
	"fmt"
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
		qsdSock: sock,
		images:  make(map[string]QCOWImage),
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
	if strings.Contains(id, "pvc-") {
		return strings.TrimPrefix(id, "pvc-")[:8]
	}

	if strings.Contains(id, "snapshot-") {
		return strings.TrimPrefix(id, "snapshot-")[:8]
	}
	return id[:8]
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
	_, err = os.Stat(i)
	if os.IsNotExist(err) {
		if err := volManager.CreateVolume(i, image.ID, strconv.FormatInt(image.Size, 10)); err != nil {
			errMessage := fmt.Sprintf("Failed creating the disk image %s:%v", image.ID, err)
			return failed(errMessage, err)
		}

	}

	_, err = os.Stat(i)
	if err != nil {
		errMessage := fmt.Sprintf("Failed stating the image %s:%v", image.ID, err)
		return failed(errMessage, err)
	}
	c.images[image.ID] = QCOWImage{
		File: i,
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

func (c *Server) DeleteVolume(ctx context.Context, image *Image) (*Response, error) {
	log.Infof("Create new monitor to delete volume")
	// Get the active layer of the image
	id, ok := c.activeLayers[image.ID]
	if !ok {
		return &Response{}, fmt.Errorf("Failed to delete the image %s: active layer not found", image.ID)
	}
	var image QCOWImage
	image, ok = c.images[id]
	if !ok {
		return &Response{}, fmt.Errof("Failed to delete the image %s: image not found", image.ID)
	}
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err)
		return failed(errMessage, err)
	}
	if err := volManager.DeleteVolume(image.QSDID); err != nil {
		errMessage := fmt.Sprintf("Cannot delete volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}

	// If there are no snapshot we can remove the backing file and the entire directory
	if image.RefCount == 0 {
		dir := fmt.Sprintf("%s/%s", imagesDir, image.ID)
		if err := os.RemoveAll(dir); err != nil {
			errMessage := fmt.Sprintf("Cannot delete image directory for the volume %s: %v", image.ID, err)
			return failed(errMessage, err)
		}
	}
	delete(c.activeLayers, image.ID)
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
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	volumeToSnapshot := getActiveLayer()
	dir := fmt.Sprintf("%s/%s", imagesDir, snapshot.SourceVolumeID)
	s := fmt.Sprintf("%s/%s-%s", dir, snapshotPrefix, snapshot.ID)
	b := fmt.Sprintf("%s/%s-%s", dir, snapshotPrefix, volumeToSnapshot)
	if snapshot.SourceVolumeID == volumeToSnapshot {
		b = fmt.Sprintf("%s/%s", dir, diskImg)
	}
	if _, err := os.Stat(dir); err != nil {
		errMessage := fmt.Sprintf("Failed checking the directory for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}

	if err := volManager.CreateSnapshot(volumeToSnapshot, snapshot.ID, b, s); err != nil {
		errMessage := fmt.Sprintf("Cannot snapshot %s: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	c.images[image.ID] = QCOWImage{}

	return &Response{}, nil

}

func (c *Server) DeleteSnapshot(ctx context.Context, snapshot *Snapshot) (*Response, error) {
	log.Infof("Create new monitor to delete snapshot")
	volManager, err := NewVolumeManager(c.qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	upperLayer := ""
	if err := volManager.StreamImage(snapshot.ID, upperLayer); err != nil {
		errMessage := fmt.Sprintf("Cannot copy snapshot %s in the upper layer: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}

	dir := fmt.Sprintf("%s/%s", imagesDir, snapshot.SourceVolumeID)
	s := fmt.Sprintf("%s/%s-%s", dir, snapshotPrefix, snapshot.ID)
	if _, err := os.Stat(dir); err != nil {
		errMessage := fmt.Sprintf("Failed checking the directory for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	//	if err := volManager.DeleteVolume(snapshot.ID); err != nil {
	//		errMessage := fmt.Sprintf("Cannot delete volume %s: %v", snapshot.ID, err)
	//		return failed(errMessage, err)
	//	}

	if err := os.Remove(s); err != nil {
		errMessage := fmt.Sprintf("Cannot delete snapshot %s: %v", snapshot.ID, err)
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

func getActiveLayer() string {
	return ""
}
