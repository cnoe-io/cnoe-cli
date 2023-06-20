package lib

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . IK8sClient
type IK8sClient interface {
	Pods(namespace string) (*corev1.PodList, error)
	CRDs(group, kind, version string) (*unstructured.UnstructuredList, error)
}

type k8sClient struct {
	clientset     *kubernetes.Clientset
	dynamicclient *dynamic.DynamicClient
}

func NewK8sClient(kubeconfig string) (IK8sClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return &k8sClient{}, err
	}

	// Create the Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return k8sClient{}, err
	}

	// Create the dynamic client
	dynamicclient, err := dynamic.NewForConfig(config)
	if err != nil {
		return k8sClient{}, err
	}

	return k8sClient{
		clientset:     clientset,
		dynamicclient: dynamicclient,
	}, nil
}

func (k k8sClient) Pods(namespace string) (*corev1.PodList, error) {
	pods, err := k.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (k k8sClient) CRDs(group, kind, version string) (*unstructured.UnstructuredList, error) {
	// Retrieve the CRD list
	crdList, err := k.dynamicclient.Resource(
		schema.GroupVersionResource{
			Group:    strings.ToLower(group),
			Resource: strings.ToLower(kind),
			Version:  strings.ToLower(version),
		},
	).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return crdList, nil
}
