package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/alicefr/csi-qsd/pkg/qsd"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	qsdPidfile = "/var/run/qsd.pid"
	timeout    = 30
)

var (
	port    = flag.String("port", "", "Port to listen")
	qsdSock = "/var/run/qsd-qmp.sock"
)

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
	qmpServer, err := qsd.NewServer(qsdSock)
	if err != nil {
		log.Fatalf("Starting connection with the QMP: %v", err)
	}
	qsd.RegisterQsdServiceServer(srv, qmpServer)
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
