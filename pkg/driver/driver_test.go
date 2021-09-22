package driver

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/kubernetes-csi/csi-test/v4/pkg/sanity"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type idGenerator struct{}

func (g *idGenerator) GenerateUniqueValidVolumeID() string {
	return uuid.New().String()
}

func (g *idGenerator) GenerateInvalidVolumeID() string {
	return g.GenerateUniqueValidVolumeID()
}

func (g *idGenerator) GenerateUniqueValidNodeID() string {
	return strconv.Itoa(1)
}

func (g *idGenerator) GenerateInvalidNodeID() string {
	return "not-an-integer"
}

func TestDriverSuite(t *testing.T) {
	socket := "/tmp/csi.sock"
	endpoint := "unix://" + socket
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove unix domain socket file %s, error: %s", socket, err)
	}

	driver := &Driver{
		name:     DefaultDriverName,
		endpoint: endpoint,
		log:      logrus.New().WithField("test", "test driver"),
	}

	ctx, cancel := context.WithCancel(context.Background())

	var eg errgroup.Group
	eg.Go(func() error {
		t.Log("Start the driver")
		return driver.Run(ctx)
	})

	cfg := sanity.NewTestConfig()
	t.Log("Cleanup")
	if err := os.RemoveAll(cfg.TargetPath); err != nil {
		t.Fatalf("failed to delete target path %s: %s", cfg.TargetPath, err)
	}
	cfg.Address = endpoint
	cfg.IDGen = &idGenerator{}
	cfg.IdempotentCount = 5
	sanity.Test(t, cfg)

	cancel()
	if err := eg.Wait(); err != nil {
		t.Errorf("driver run failed: %s", err)
	}
}
