package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	qsdSock    = "/var/run/qsd-qmp.sock"
	qsdPidfile = "/var/run/qsd.pid"
	imagesDir  = "/var/run/images"
	socketDir  = "/var/run/sockets"
	timeout    = 30
	diskImg    = "disk.img"
	vhostSock  = "vhost.sock"
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

	log.Fatalln(srv.Serve(lis))
}

func (c *server) CreateVolume(ctx context.Context, image *qsd.Image) (*qsd.Response, error) {
	log.Infof("Create new Monitor")
	volManager, err := qsd.NewVolumeManager(qsdSock)
	if err != nil {
		return &qsd.Response{
			Success: false,
			Message: fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err),
		}, nil
	}
	dir := fmt.Sprintf("%s/%s", imagesDir, image.ID)
	i := fmt.Sprintf("%s/%s", imagesDir, diskImg)
	// Create directory for the volume if it doesn't exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &qsd.Response{
			Success: false,
			Message: fmt.Sprintf("Cannot create directory for the volume:%s", image.ID),
		}, nil
	}
	_, err = os.Stat(i)
	if os.IsNotExist(err) {
		if err := volManager.CreateVolume(i, image.ID, *image.Size); err != nil {
			return &qsd.Response{
				Success: false,
				Message: fmt.Sprintf("Failed creating the disk image %s:%v", image.ID, err),
			}, nil
		}

	}

	_, err = os.Stat(i)
	if err != nil {
		return &qsd.Response{
			Success: false,
			Message: fmt.Sprintf("Failed stating the image %s:%v", image.ID, err),
		}, nil

	}

	return &qsd.Response{
		Success: true,
	}, nil
}

func (c *server) ExposeVhostUser(ctx context.Context, image *qsd.Image) (*qsd.Response, error) {
	log.Infof("Create new Monitor")
	volManager, err := qsd.NewVolumeManager(qsdSock)
	if err != nil {
		return &qsd.Response{
			Success: false,
			Message: fmt.Sprintf("Failed creating the qsd monitor fol vol %s:%v", image.ID, err),
		}, nil
	}
	dir := fmt.Sprintf("%s/%s", socketDir, image.ID)
	// Create directory for the socket if it doesn't exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &qsd.Response{
			Success: false,
			Message: fmt.Sprintf("Cannot create socket directory for the volume %s: %v", image.ID, err),
		}, nil
	}
	socket := fmt.Sprintf("%s/%s", dir, vhostSock)
	// Expose and create vhost-user socket
	if _, err := os.Stat(socket); os.IsExist(err) {
		if err := os.Remove(socket); err != nil {
			return &qsd.Response{
				Success: false,
				Message: fmt.Sprintf("Cannot create socket directory for the volume %s: %v", image.ID, err),
			}, nil
		}
	}

	if err := volManager.ExposeVhostUser(image.ID, socket); err != nil {
		return &qsd.Response{
			Success: false,
			Message: fmt.Sprintf("Cannot create socket for the volume %s: %v", image.ID, err),
		}, nil
	}
	return &qsd.Response{
		Success: true,
	}, nil

}
