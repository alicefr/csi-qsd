package driver

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	DefaultDriverName = "qsd.csi.com"
	version           = "0.0.0"
)

const (
	SocketDir = "/var/run/qsd/sockets"
	vhostSock = "vhost.sock"
)

type Volume struct {
	id   string
	size int64
	node string
}

type Driver struct {
	csi.UnimplementedControllerServer
	csi.UnimplementedNodeServer
	name     string
	version  string
	endpoint string
	readyMu  sync.Mutex
	ready    bool
	port     string
	storage  map[string]Volume

	srv *grpc.Server
	log *logrus.Entry

	nodeId string
}

func NewDriver(endpoint, driverName, nodeId, port string) (*Driver, error) {
	log := logrus.New().WithFields(logrus.Fields{
		"endpoint": endpoint,
		"node-id":  nodeId,
	})
	return &Driver{
		version:  version,
		endpoint: endpoint,
		storage:  make(map[string]Volume),
		name:     driverName,
		log:      log,
		ready:    true,
		nodeId:   nodeId,
		port:     port,
	}, nil
}

func (d *Driver) deleteVolume(id string) {
	delete(d.storage, id)
}

func (d *Driver) Run(ctx context.Context) error {
	u, err := url.Parse(d.endpoint)
	if err != nil {
		return fmt.Errorf("unable to parse address: %q", err)
	}

	addr := path.Join(u.Host, filepath.FromSlash(u.Path))
	if u.Host == "" {
		addr = filepath.FromSlash(u.Path)
	}
	d.log.Infof("Socket %s schema %s", d.endpoint, u.Scheme)
	d.log.Infof("QSD grpc server listens at port %s", d.port)
	// CSI plugins talk only over UNIX sockets currently
	if u.Scheme != "unix" {
		return fmt.Errorf("currently only unix domain sockets are supported, have: %s", u.Scheme)
	} else {
		// remove the socket if it's already there. This can happen if we
		// deploy a new version and the socket was created from the old running
		// plugin.
		d.log.WithField("socket", addr).Info("removing socket")
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove unix domain socket file %s, error: %s", addr, err)
		}
	}

	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// log response errors for better observability
	errHandler := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			d.log.WithError(err).WithField("method", info.FullMethod).Error("method failed")
		}
		return resp, err
	}

	d.srv = grpc.NewServer(grpc.UnaryInterceptor(errHandler))
	csi.RegisterIdentityServer(d.srv, d)
	csi.RegisterControllerServer(d.srv, d)
	csi.RegisterNodeServer(d.srv, d)

	d.log.WithField("addr", addr).Info("server started")
	return d.srv.Serve(listener)
}

// Stop stops the plugin
func (d *Driver) Stop() {
	d.log.Info("server stopped")
	d.srv.Stop()
}
