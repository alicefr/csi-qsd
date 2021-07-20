package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alicefr/csi-qsd/pkg/driver"
)

func main() {
	var (
		endpoint   = flag.String("endpoint", "unix:///var/lib/kubelet/plugins/"+driver.DefaultDriverName+"/csi.sock", "CSI endpoint")
		nodeId     = flag.String("node-id", "", "Specify node id where the plugin runs")
		driverName = flag.String("driver-name", driver.DefaultDriverName, "Name for the driver")
		port       = flag.String("port", "", "Port for the qsd grpc server")
		help       = flag.Bool("help", false, "Print help and exit")
	)
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	drv, err := driver.NewDriver(*endpoint, *driverName, *nodeId, *port)
	if err != nil {
		log.Fatalln(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	if err := drv.Run(ctx); err != nil {
		log.Fatalln(err)
	}
}
