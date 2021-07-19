package driver

import (
	"context"
	"fmt"
	"github.com/alicefr/csi-qsd/pkg/qsd"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
)

const DefaultDriverName = "qsd.csi.com"
const version = "0.0.0"

type Driver struct {
	name     string
	version  string
	readyMu  sync.Mutex
	ready    bool
	endpoint string
	storage  map[string]*qsd.Volume
	// Path to unix QMP socket to talk to QSD
	qmp string

	srv            *grpc.Server
	log            *logrus.Entry
	pathQsdVolumes string
	nodeId         string
	k8sclient      *client.K8sClient
}

func NewDriver(endpoint, driverName, sc, nodeId, dir, qmp string) (*Driver, error) {
	log := logrus.New().WithFields(logrus.Fields{
		"endpoint": endpoint,
		"sc":       sc,
		"node-id":  nodeId,
	})
	c, err := client.NewK8sClientFromCluster()
	if err != nil {
		return &Driver{}, fmt.Errorf("faild creating the k8s client: %v", err)
	}
	log.Info("create k8s client")
	return &Driver{
		version:        version,
		endpoint:       endpoint,
		storage:        make(map[string]*qsd.Volume),
		name:           driverName,
		log:            log,
		ready:          true,
		pathQsdVolumes: dir,
		nodeId:         nodeId,
		k8sclient:      c,
		qmp:            qmp,
	}, nil
}

func indexOf(element string, data []qsd.Volume) int {
	for k, v := range data {
		if element == v.ID {
			return k
		}
	}
	return -1
}

func (d *Driver) deleteVolume(id string) {
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
	//	csi.RegisterControllerServer(d.srv, d)
	csi.RegisterNodeServer(d.srv, d)

	d.log.WithField("addr", addr).Info("server started")
	return d.srv.Serve(listener)
}

// Stop stops the plugin
func (d *Driver) Stop() {
	d.log.Info("server stopped")
	d.srv.Stop()
}
