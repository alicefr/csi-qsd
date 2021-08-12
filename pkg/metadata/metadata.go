package metadata

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type MetadataServer struct {
	MetadataServiceServer
	client *K8sClient
}

type K8sClient struct {
	client *kubernetes.Clientset
}

func newK8sClient(config *rest.Config) (*K8sClient, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return &K8sClient{}, err
	}

	return &K8sClient{
		client: clientset,
	}, nil
}

// NewK8sClientFromCluster creates a k8s client running inside the cluster.
// This method must be called outside a cluster
func NewK8sClientFromCluster() (*K8sClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return &K8sClient{}, err
	}

	return newK8sClient(config)
}

// NewK8sClientFromConfig creates a k8s client from a given configuration.
// This method must be called outside a cluster
func NewK8sClientFromConfig(kubeconfig string) (*K8sClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return &K8sClient{}, err
	}

	return newK8sClient(config)
}

func NewMetadataServer() (*MetadataServer, error) {
	client, err := NewK8sClientFromCluster()
	if err != nil {
		return nil, err
	}
	return &MetadataServer{
		client: client,
	}, err
}

func (s *MetadataServer) GetVolumes(context.Context, *Node) (*ResponseMetadata, error) {
	return nil, nil
}

func (s *MetadataServer) AddMetadata(context.Context, *Metadata) (*ResponseMetadata, error) {
	return nil, nil
}
