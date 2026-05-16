package watcher

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1typed "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1typed "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingv1typed "k8s.io/client-go/kubernetes/typed/networking/v1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/tarzaa1/kubeinsights/pkg/event"
)

type Watcher struct {
	core      corev1typed.CoreV1Interface
	apps      appsv1typed.AppsV1Interface
	network   networkingv1typed.NetworkingV1Interface
	metrics   *metrics.Clientset
	events    chan event.Event
	namespace string
	log       *log.Logger
	errLog    *log.Logger
}

func New(
	core corev1typed.CoreV1Interface,
	apps appsv1typed.AppsV1Interface,
	network networkingv1typed.NetworkingV1Interface,
	metricsClient *metrics.Clientset,
	events chan event.Event,
	namespace string,
	infoLog *log.Logger,
	errLog *log.Logger,
) *Watcher {
	return &Watcher{
		core:      core,
		apps:      apps,
		network:   network,
		metrics:   metricsClient,
		events:    events,
		namespace: namespace,
		log:       infoLog,
		errLog:    errLog,
	}
}

func (w *Watcher) send(e event.Event) {
	w.log.Printf("%s %s %s", e.Id, e.Action, e.Kind)
	w.events <- e
}

// Snapshot lists all current cluster resources and emits them as Add events.
// It must be called before Watch.
func (w *Watcher) Snapshot(ctx context.Context) error {
	w.send(event.New("Add", "Cluster", nil))

	nodes, err := w.core.Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing nodes: %w", err)
	}
	w.log.Printf("found %d nodes", len(nodes.Items))
	for _, node := range nodes.Items {
		w.send(event.New("Add", "Node", event.ToJSON(node)))
	}

	configmaps, err := w.core.ConfigMaps(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing configmaps: %w", err)
	}
	w.log.Printf("found %d configmaps", len(configmaps.Items))
	for _, cm := range configmaps.Items {
		w.send(event.New("Add", "ConfigMap", event.ToJSON(cm)))
	}

	deployments, err := w.apps.Deployments(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing deployments: %w", err)
	}
	w.log.Printf("found %d deployments", len(deployments.Items))
	for _, d := range deployments.Items {
		w.send(event.New("Add", "Deployment", event.ToJSON(d)))
	}

	replicasets, err := w.apps.ReplicaSets(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing replicasets: %w", err)
	}
	w.log.Printf("found %d replicasets", len(replicasets.Items))
	for _, rs := range replicasets.Items {
		w.send(event.New("Add", "ReplicaSet", event.ToJSON(rs)))
	}

	pods, err := w.core.Pods(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}
	w.log.Printf("found %d pods", len(pods.Items))
	for _, pod := range pods.Items {
		w.send(event.New("Add", "Pod", event.ToJSON(pod)))
	}

	services, err := w.core.Services(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}
	w.log.Printf("found %d services", len(services.Items))
	for _, svc := range services.Items {
		w.send(event.New("Add", "Service", event.ToJSON(svc)))
	}

	ingresses, err := w.network.Ingresses(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing ingresses: %w", err)
	}
	w.log.Printf("found %d ingresses", len(ingresses.Items))
	for _, ing := range ingresses.Items {
		w.send(event.New("Add", "Ingress", event.ToJSON(ing)))
	}

	return nil
}
