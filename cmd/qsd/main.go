package main

import (
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
	timeout    = 30
)

var (
	port = flag.String("port", "", "Port to listen")
)

type server struct {
	qsd.UnimplementedQsdServiceServer
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
