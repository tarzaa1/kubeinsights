package watcher

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tarzaa1/kubeinsights/pkg/event"
)

// PollMetrics continuously polls node and pod metrics, emitting Update events.
// It exits when ctx is cancelled. Run in a goroutine.
func (w *Watcher) PollMetrics(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.pollNodeMetrics(ctx); err != nil {
			w.errLog.Printf("node metrics: %v — retrying in 10s", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
			}
			continue
		}

		if err := w.pollPodMetrics(ctx); err != nil {
			w.errLog.Printf("pod metrics: %v — retrying in 10s", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (w *Watcher) pollNodeMetrics(ctx context.Context) error {
	nodeMetrics, err := w.metrics.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	w.send(event.New("Update", "NodeMetrics", event.ToJSON(nodeMetrics)))
	return nil
}

func (w *Watcher) pollPodMetrics(ctx context.Context) error {
	podMetrics, err := w.metrics.MetricsV1beta1().PodMetricses(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	w.send(event.New("Update", "PodMetrics", event.ToJSON(podMetrics)))
	return nil
}
