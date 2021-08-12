package metadata

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	nodeLabel         = "node"
	prefixAnn         = "csi-qsd"
	annID             = prefixAnn + "id"
	annQSDID          = prefixAnn + "qsdID"
	annBackingImageID = prefixAnn + "backingImageID"
	annRefCount       = prefixAnn + "refCount"
)

type MetadataServer struct {
	MetadataServiceServer
	client *kubernetes.Clientset
}

func newK8sClient(config *rest.Config) (*kubernetes.Clientset, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

// NewK8sClientFromCluster creates a k8s client running inside the cluster.
// This method must be called outside a cluster
func NewK8sClientFromCluster() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return newK8sClient(config)
}

// NewK8sClientFromConfig creates a k8s client from a given configuration.
// This method must be called outside a cluster
func NewK8sClientFromConfig(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
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

func parseAnnotationsFromPVtoMetadata(pv *corev1.PersistentVolume) *Metadata {
	id, okID := pv.ObjectMeta.Annotations[annID]
	qsdID, okQsdID := pv.ObjectMeta.Annotations[annQSDID]
	bID, okBID := pv.ObjectMeta.Annotations[annBackingImageID]
	r, okR := pv.ObjectMeta.Annotations[annRefCount]
	if okID && okBID && okQsdID && okR {
		refCount, err := strconv.ParseInt(r, 10, 32)
		if err != nil {
			return err
		}
		return Metadata{
			ID:             id,
			QSDID:          qsdID,
			RefCount:       refCount,
			BackingImageID: bID,
		}
	}
	return nil
}

func (s *MetadataServer) GetVolumes(ctx context.Context, node *Node) (*ResponseGetVolumes, error) {
	// Select PVs with the label of the node
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{nodeLabel: node.GetNodeID()}}
	options := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	pvList, err := s.client.CoreV1().PersistentVolumes().List(context.TODO(), options)
	if err != nil {
		return nil, err
	}
	var metadata []Metadata
	// Parse the annotations with the metadata information
	for pv, _ := range pvList.Items {
		if m := parseAnnotationsFromPVtoMetadata; m != nil {
			metadata = append(metadata, m)
		}
	}
	return m, nil
}

func (s *MetadataServer) AddMetadata(context.Context, *Metadata) (*ResponseAddMetadata, error) {
	// Add label to the PV with the node where it has be created
	// Add annotation from the metadata
	return nil, nil
}
