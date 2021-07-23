package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	qsdSock        = "/var/run/qsd-qmp.sock"
	qsdPidfile     = "/var/run/qsd.pid"
	imagesDir      = "/var/run/qsd/images"
	socketDir      = "/var/run/qsd/sockets"
	timeout        = 30
	diskImg        = "disk.img"
	vhostSock      = "vhost.sock"
	snapshotPrefix = "snap"
)

var (
	port = flag.String("port", "", "Port to listen")
)

type server struct {
	qsd.QsdServiceServer
}

type Result struct {
	Error  error
	Output string
}

func main() {
	flag.Parse()
	if *port == "" {
		log.Fatalf("Specify the port")
	}
	log.Infof("Server listening at %s", *port)
	r := make(chan Result, 1)
	// Start the qsd in a separate go routine
	go func() {
		cmd := exec.Command("qemu-storage-daemon", "--pidfile",
			qsdPidfile,
			"--chardev",
			fmt.Sprintf("socket,server=on,path=%s,id=chardev0,wait=off", qsdSock),
			"--monitor",
			"chardev=chardev0")
		out, err := cmd.CombinedOutput()
		r <- Result{Error: err, Output: string(out)}

	}()
	waitForQSDPidfile := make(chan bool)
	go func() {
		for {
			if _, err := os.Stat(qsdPidfile); !os.IsNotExist(err) {
				waitForQSDPidfile <- true
			}
			time.Sleep(50 * time.Millisecond)
		}

	}()

	select {
	case <-waitForQSDPidfile:
		log.Infof("QSD started...")
	case result := <-r:
		// the qsd terminated check the error and output
		log.Fatalf("QSD terminated output: %s err: %v", result.Output, result.Error)
	case <-time.After(time.Second * timeout):
		log.Fatalf("Timeout in waiting for the qsd to start")

	}
	// List for grpc command
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", *port))
	if err != nil {
		log.Fatalln(err)
	}
	srv := grpc.NewServer()
	qsd.RegisterQsdServiceServer(srv, &server{})
	serError := make(chan error, 1)
	go func() {
		serError <- srv.Serve(lis)
	}()
	ch := make(chan os.Signal, 10)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGHUP)
	select {
	case <-ch:
		log.Info("Gracefully terminate")
		srv.Stop()
	case result := <-r:
		// the qsd terminated check the error and output
		log.Fatalf("QSD terminated output: %s err: %v", result.Output, result.Error)
	case result := <-serError:
		log.Fatalf("GRPC server failed: %v", result)
	}
}

func failed(m string, err error) (*qsd.Response, error) {
	log.Errorf(m)
	return &qsd.Response{
		Success: false,
		Message: m,
	}, err

}

func (c *server) CreateVolume(ctx context.Context, image *qsd.Image) (*qsd.Response, error) {
	log.Infof("Create new monitor for the volume creation")
	volManager, err := qsd.NewVolumeManager(qsdSock)
	defer volManager.Disconnect()
	if err != nil {
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

	return &qsd.Response{
		Success: true,
	}, nil
}

func (c *server) ExposeVhostUser(ctx context.Context, image *qsd.Image) (*qsd.Response, error) {
	log.Infof("Create new monitor to expose vhost user")
	volManager, err := qsd.NewVolumeManager(qsdSock)
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
	return &qsd.Response{
		Success: true,
	}, nil

}

func (c *server) DeleteVolume(ctx context.Context, image *qsd.Image) (*qsd.Response, error) {
	log.Infof("Create new monitor to delete volume")
	volManager, err := qsd.NewVolumeManager(qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err)
		return failed(errMessage, err)
	}
	if err := volManager.DeleteVolume(image.ID); err != nil {
		errMessage := fmt.Sprintf("Cannot delete volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}

	dir := fmt.Sprintf("%s/%s", imagesDir, image.ID)
	if err := os.RemoveAll(dir); err != nil {
		errMessage := fmt.Sprintf("Cannot delete image directory for the volume %s: %v", image.ID, err)
		return failed(errMessage, err)
	}
	return &qsd.Response{}, nil
}

func (c *server) DeleteExporter(ctx context.Context, image *qsd.Image) (*qsd.Response, error) {
	log.Infof("Create new monitor to delete exporter")
	volManager, err := qsd.NewVolumeManager(qsdSock)
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
	return &qsd.Response{}, nil
}

func (c *server) CreateSnapshot(ctx context.Context, snapshot *qsd.Snapshot) (*qsd.Response, error) {
	log.Infof("Create new monitor to snapshot")
	volManager, err := qsd.NewVolumeManager(qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, snapshot.SourceVolumeID)
	s := fmt.Sprintf("%s/%s-%s", dir, snapshotPrefix, snapshot.ID)
	if _, err := os.Stat(dir); err != nil {
		errMessage := fmt.Sprintf("Failed checking the directory for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}

	if err := volManager.CreateSnapshot(snapshot.VolumeToSnapshot, snapshot.ID, s); err != nil {
		errMessage := fmt.Sprintf("Cannot snapshot %s: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}

	return &qsd.Response{}, nil

}

func (c *server) DeleteSnapshot(ctx context.Context, snapshot *qsd.Snapshot) (*qsd.Response, error) {
	log.Infof("Create new monitor to snapshot")
	volManager, err := qsd.NewVolumeManager(qsdSock)
	defer volManager.Disconnect()
	if err != nil {
		errMessage := fmt.Sprintf("Failed creating the qsd monitor for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, snapshot.SourceVolumeID)
	s := fmt.Sprintf("%s/%s-%s", dir, snapshotPrefix, snapshot.ID)
	if _, err := os.Stat(dir); err != nil {
		errMessage := fmt.Sprintf("Failed checking the directory for snapshot %s:%v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	// TODO find right way how to remove a snapshot
	//	if err := volManager.DeleteVolume(snapshot.ID); err != nil {
	//		errMessage := fmt.Sprintf("Cannot delete volume %s: %v", snapshot.ID, err)
	//		return failed(errMessage, err)
	//	}

	if err := os.Remove(s); err != nil {
		errMessage := fmt.Sprintf("Cannot delete snapshot %s: %v", snapshot.ID, err)
		return failed(errMessage, err)
	}
	return &qsd.Response{}, nil
}
