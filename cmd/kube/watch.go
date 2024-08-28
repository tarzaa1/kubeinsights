package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tarzaa1/kubeinsights/pkg/kubestate"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ResourceToJSON[T any](resource T) json.RawMessage {
	jsonBytes, err := json.MarshalIndent(resource, "", "    ")
	if err != nil {
		panic(err.Error())
	}
	return jsonBytes
}

func main() {

	k8sclient := kubestate.K8sClientSet()

	namespace := "default"

	watcher, err := k8sclient.CoreV1().Events(namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	currentTime := time.Now()

	for event := range watcher.ResultChan() {
		item := event.Object.(*v1.Event)

		if item.CreationTimestamp.Time.Before(currentTime) {
			continue
		}

		fmt.Println(string(ResourceToJSON(event)))
		// switch event.Type {
		// case watch.Added:
		// 	if item.InvolvedObject.Kind == "Pod" {
		// 		switch item.Reason {
		// 		case "Killing":
		// 			log.Printf("\nKubernetes API Server: Killing Pod %s\n", item.InvolvedObject.Name)
		// 		case "Started":
		// 			time.Sleep(3 * time.Second)
		// 			log.Printf("\nKubernetes API Server: Started Pod %s\n", item.InvolvedObject.Name)
		// 		}
		// 	}

		// 	if item.InvolvedObject.Kind == "Node" {
		// 		switch item.Reason {
		// 		case "RemovingNode":
		// 			log.Printf("\nKubernetes API Server: Removing Node %s\n", item.InvolvedObject.Name)
		// 		case "Starting":
		// 			log.Printf("\nKubernetes API Server: Starting Node %s\n", item.InvolvedObject.Name)
		// 		}
		// 	}
		// case watch.Modified:
		// case watch.Deleted:
		// }
	}
}
