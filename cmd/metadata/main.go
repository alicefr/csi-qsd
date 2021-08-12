package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/alicefr/csi-qsd/pkg/metadata"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	port = flag.String("port", "", "Port to listen")
)

func main() {
	flag.Parse()
	if *port == "" {
		log.Fatalf("Specify the port")
	}
	log.Infof("Server listening at %s", *port)
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", *port))
	if err != nil {
		log.Fatalln(err)
	}
	srv := grpc.NewServer()
	metadataServer, err := metadata.NewMetadataServer()
	if err != nil {
		log.Fatalf("Failed creating the metadata server: %v", err)
	}
	metadata.RegisterMetadataServiceServer(srv, metadataServer)
	srv.Serve(lis)

}
