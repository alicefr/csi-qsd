package driver

import (
	"context"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestDriver_NodePublishVolume(t *testing.T) {
	ctx := context.Background()
	d := &Driver{
		log:            logrus.New().WithField("test", true),
		pathQsdVolumes: "/tmp/test-volumes",
	}

	req := &csi.NodePublishVolumeRequest{
		VolumeId:         "testid",
		TargetPath:       "/tmp/test-target-path",
		VolumeCapability: &csi.VolumeCapability{},
	}
	if err := os.Mkdir(req.TargetPath, 0755); err != nil {
		t.Fatalf("Faild in creating test dir %s: %v", req.TargetPath, err)
		removeDirs(req.TargetPath)
	}

	_, err := d.NodePublishVolume(ctx, req)
	if err != nil {
		removeDirs(req.TargetPath, d.pathQsdVolumes)
		t.Logf("Driver.NodePublishVolume failed with error = %v", err)
		t.Fail()
	}
	if err := unmount(req.TargetPath); err != nil {
		t.Fatalf("Faild cleaning up the test dir %s: %v", req.TargetPath, err)
	}
	removeDirs(req.TargetPath, d.pathQsdVolumes)
}
