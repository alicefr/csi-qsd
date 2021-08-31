package metadata

import (
	"context"
	"fmt"
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

func parseAnnotationsFromPVtoMetadata(pv corev1.PersistentVolume) (*Metadata, error) {
	var err error
	var refCount int64
	id, okID := pv.ObjectMeta.Annotations[annID]
	qsdID, okQsdID := pv.ObjectMeta.Annotations[annQSDID]
	bID, okBID := pv.ObjectMeta.Annotations[annBackingImageID]
	if !okID {
		return nil, fmt.Errorf("Annotation %s not found", annID)
	}
	if !okQsdID {
		return nil, fmt.Errorf("Annotation %s not found", okQsdID)
	}
	if !okBID {
		bID = ""
	}
	r, okR := pv.ObjectMeta.Annotations[annRefCount]
	if okR {
		refCount, err = strconv.ParseInt(r, 10, 32)
		if err != nil {
			return nil, err
		}
	}
	return &Metadata{
		ID:             id,
		QSDID:          qsdID,
		RefCount:       uint32(refCount),
		BackingImageID: bID,
	}, nil
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
	var metadata []*Metadata
	// Parse the annotations with the metadata information
	for _, pv := range pvList.Items {
		m, err := parseAnnotationsFromPVtoMetadata(pv)
		if m != nil && err == nil {
			metadata = append(metadata, m)
		}
	}
	return &ResponseGetVolumes{
		Volumes: metadata,
	}, nil
}

func (s *MetadataServer) AddMetadata(context.Context, *Metadata) (*ResponseAddMetadata, error) {
	// Add label to the PV with the node where it has be created
	// Add annotation from the metadata
	return nil, nil
}
