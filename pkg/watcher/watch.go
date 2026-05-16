package watcher

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/tarzaa1/kubeinsights/pkg/event"
)

// Watch monitors the cluster event stream and emits Add/Delete events for pods
// and nodes. It blocks until a "done" sentinel is received or the context is
// cancelled.
func (w *Watcher) Watch(ctx context.Context) {
	initialList, err := w.core.Events(w.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		w.errLog.Printf("listing events for watch: %v", err)
		return
	}
	resourceVersion := initialList.ResourceVersion

	for {
		watcher, err := w.core.Events(w.namespace).Watch(ctx, metav1.ListOptions{
			ResourceVersion: resourceVersion,
			Watch:           true,
		})
		if err != nil {
			w.errLog.Printf("creating watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		done := w.processEvents(ctx, watcher.ResultChan(), &resourceVersion)
		watcher.Stop()

		if done {
			return
		}

		w.log.Println("watch channel closed, re-establishing...")
		list, err := w.core.Events(w.namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			w.errLog.Printf("listing events after watch closed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		resourceVersion = list.ResourceVersion
	}
}

// processEvents handles a single watch session. Returns true if a "done"
// sentinel was received and Watch should stop.
func (w *Watcher) processEvents(ctx context.Context, ch <-chan watch.Event, resourceVersion *string) (done bool) {
	for watchEvent := range ch {
		switch watchEvent.Type {
		case watch.Error:
			status, _ := watchEvent.Object.(*metav1.Status)
			w.errLog.Printf("watch error: %+v", status)
			return false

		case watch.Added:
			k8sEvent, ok := watchEvent.Object.(*v1.Event)
			if !ok {
				continue
			}
			*resourceVersion = k8sEvent.ResourceVersion

			switch k8sEvent.InvolvedObject.Kind {
			case "Pod":
				if w.handlePodEvent(ctx, k8sEvent) {
					return true
				}
			case "Node":
				w.handleNodeEvent(ctx, k8sEvent)
			}
		}
	}
	return false
}

func (w *Watcher) handlePodEvent(ctx context.Context, k8sEvent *v1.Event) (done bool) {
	podName := k8sEvent.InvolvedObject.Name
	switch k8sEvent.Reason {
	case "Killing":
		w.log.Printf("pod deleted: %s", podName)
		w.send(event.New("Delete", "Pod", event.ToJSON(podName)))

	case "Started":
		if podName == "done" {
			w.send(event.New("Add", "Done", event.ToJSON("")))
			return true
		}
		time.Sleep(3 * time.Second)
		w.log.Printf("pod started: %s", podName)
		w.sendPodWithImage(ctx, podName)
	}
	return false
}

func (w *Watcher) handleNodeEvent(ctx context.Context, k8sEvent *v1.Event) {
	nodeName := k8sEvent.InvolvedObject.Name
	switch k8sEvent.Reason {
	case "RemovingNode":
		w.log.Printf("node removing: %s", nodeName)
		w.send(event.New("Delete", "Node", event.ToJSON(nodeName)))

	case "Starting":
		w.log.Printf("node starting: %s", nodeName)
		node, err := w.core.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			w.errLog.Printf("fetching node %s: %v", nodeName, err)
			return
		}
		w.send(event.New("Add", "Node", event.ToJSON(*node)))
	}
}

// sendPodWithImage fetches a pod and the image it introduced on its node,
// then emits both as Add events.
func (w *Watcher) sendPodWithImage(ctx context.Context, podName string) {
	pod, err := w.fetchPod(ctx, podName)
	if err != nil {
		w.errLog.Printf("fetching pod %s: %v", podName, err)
		return
	}

	if err := w.sendNodeImage(ctx, pod); err != nil {
		w.errLog.Printf("sending image for pod %s: %v", podName, err)
	}

	w.send(event.New("Add", "Pod", event.ToJSON(*pod)))
}

func (w *Watcher) fetchPod(ctx context.Context, name string) (*v1.Pod, error) {
	var pod *v1.Pod
	var err error
	for tries := 0; tries < 5; tries++ {
		pod, err = w.core.Pods(w.namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return pod, nil
		}
		time.Sleep(1 * time.Second)
	}
	return nil, err
}

func (w *Watcher) sendNodeImage(ctx context.Context, pod *v1.Pod) error {
	if len(pod.Status.ContainerStatuses) == 0 {
		return nil
	}
	node, err := w.core.Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	imageID := pod.Status.ContainerStatuses[0].ImageID
	for _, image := range node.Status.Images {
		for _, name := range image.Names {
			if name == imageID {
				w.send(event.New("Add", "Image", event.ToJSON(event.Image{
					NodeUID: string(node.GetUID()),
					Data:    image,
				})))
				return nil
			}
		}
	}
	return nil
}
