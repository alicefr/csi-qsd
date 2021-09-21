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
	NodeLabel         = "node"
	prefixAnn         = "csi-qsd"
	AnnID             = prefixAnn + "/id"
	AnnQSDID          = prefixAnn + "/qsdID"
	AnnBackingImageID = prefixAnn + "/backingImageID"
	AnnRefCount       = prefixAnn + "/refCount"
)

type MetadataServer struct {
	MetadataServiceServer
	Client kubernetes.Interface
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
		Client: client,
	}, err
}

func parseAnnotationsFromPVtoMetadata(pv corev1.PersistentVolume) (*Metadata, error) {
	var err error
	var refCount int64
	id, okID := pv.ObjectMeta.Annotations[AnnID]
	qsdID, okQsdID := pv.ObjectMeta.Annotations[AnnQSDID]
	bID, okBID := pv.ObjectMeta.Annotations[AnnBackingImageID]
	if !okID {
		return nil, fmt.Errorf("Annotation %s not found", AnnID)
	}
	if !okQsdID {
		return nil, fmt.Errorf("Annotation %s not found", AnnQSDID)
	}
	if !okBID {
		bID = ""
	}
	r, okR := pv.ObjectMeta.Annotations[AnnRefCount]
	if okR {
		refCount, err = strconv.ParseInt(r, 10, 32)
		if err != nil {
			return nil, err
		}
	}
	return &Metadata{
		ID:             id,
		QSDID:          qsdID,
		RefCount:       refCount,
		BackingImageID: bID,
	}, nil
}

func (s *MetadataServer) GetVolumes(ctx context.Context, node *Node) (*ResponseGetVolumes, error) {
	// Select PVs with the label of the node
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{NodeLabel: node.GetNodeID()}}
	options := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	pvList, err := s.Client.CoreV1().PersistentVolumes().List(context.TODO(), options)
	if err != nil {
		return nil, err
	}
	var metadata []*Metadata
	// Parse the Annotations with the metadata information
	for _, pv := range pvList.Items {
		m, err := parseAnnotationsFromPVtoMetadata(pv)
		if m != nil && err == nil {
			m.Node = node.NodeID
			metadata = append(metadata, m)
		}
	}
	return &ResponseGetVolumes{
		Volumes: metadata,
	}, nil
}

func (s *MetadataServer) AddMetadata(_ context.Context, m *Metadata) (*ResponseAddMetadata, error) {
	pv, err := s.Client.CoreV1().PersistentVolumes().Get(context.TODO(), m.ID, metav1.GetOptions{})
	if err != nil {
		return &ResponseAddMetadata{}, err
	}
	if pv.ObjectMeta.Labels == nil {
		pv.ObjectMeta.Labels = make(map[string]string)
	}

	v, ok := pv.ObjectMeta.Labels[NodeLabel]
	if !ok {
		pv.ObjectMeta.Labels[NodeLabel] = m.Node
	} else if v != m.Node {
		return &ResponseAddMetadata{}, fmt.Errorf("Requested node %s and node label on the PV %s don't match", v, m.Node)
	}
	if pv.ObjectMeta.Annotations == nil {
		pv.ObjectMeta.Annotations = make(map[string]string)
	}

	pv.ObjectMeta.Annotations[AnnID] = m.ID
	pv.ObjectMeta.Annotations[AnnQSDID] = m.QSDID
	pv.ObjectMeta.Annotations[AnnBackingImageID] = m.BackingImageID
	pv.ObjectMeta.Annotations[AnnRefCount] = strconv.FormatInt(m.RefCount, 10)

	if _, err := s.Client.CoreV1().PersistentVolumes().Update(context.TODO(), pv, metav1.UpdateOptions{}); err != nil {
		return &ResponseAddMetadata{}, err
	}

	return &ResponseAddMetadata{}, nil
}
