module github.com/alicefr/csi-qsd

go 1.15

require (
	github.com/container-storage-interface/spec v1.3.0
	github.com/digitalocean/go-libvirt v0.0.0-20200810224808-b9c702499bf7 // indirect
	github.com/digitalocean/go-qemu v0.0.0-20200529005954-1b453d036a9c
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.1.2
	github.com/kubernetes-csi/csi-test/v4 v4.0.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/xlab/treeprint v1.1.0
	golang.org/x/net v0.0.0-20210716203947-853a461950ff // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	google.golang.org/genproto v0.0.0-20210716133855-ce7ef5c701ea // indirect
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
)

replace github.com/alicefr/csi-qsd => ./
