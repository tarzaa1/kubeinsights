package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	// publisher "kubeinsights/pkg/hedera"

	publisher "kubeinsights/pkg/kafka"
	"kubeinsights/pkg/kubestate"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics"
)

type Event struct {
	Id        uuid.UUID       `json:"id"`
	Timestamp string          `json:"timestamp"`
	Action    string          `json:"action"`
	Kind      string          `json:"kind"`
	Body      json.RawMessage `json:"body"`
}

func NewEvent(action, kind string, body json.RawMessage) Event {
	return Event{
		Id:     uuid.New(),
		Action: action,
		Kind:   kind,
		Body:   body,
	}
}

func (e *Event) SetTimestamp(timestamp string) {
	e.Timestamp = timestamp
}

func (e *Event) MarshalToJSON() []byte {
	message, err := json.Marshal(e)
	if err != nil {
		panic(err.Error())
	}
	return message
}

type Image struct {
	NodeUID string            `json:"nodeUID"`
	Data    v1.ContainerImage `json:"data"`
}

func ResourceToJSON[T any](resource T) json.RawMessage {
	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		panic(err.Error())
	}
	return jsonBytes
}

func Send(client publisher.Client, topicID string, event Event) {
	send_status := publisher.SubmitMessage(client, topicID, event.MarshalToJSON())
	fmt.Printf("%s %s", event.Id, send_status)
}

func worker(client publisher.Client, topicID string, events <-chan Event, wg *sync.WaitGroup) {
	defer wg.Done()
	for event := range events {
		Send(client, topicID, event)
	}
}

func metricsWorker(k8sRESTClient rest.Interface, events chan<- Event) {
	nodeMetricsList := metrics.NodeMetricsList{}
	for {
		data, err := k8sRESTClient.Get().AbsPath("apis/metrics.k8s.io/v1beta1/nodes").DoRaw(context.TODO())
		if err != nil {
			panic(err.Error())
		}
		// fmt.Print(string(data))
		err = json.Unmarshal(data, &nodeMetricsList)
		if err != nil {
			panic(err.Error())
		}

		for _, nodeMetrics := range nodeMetricsList.Items {
			event := NewEvent("Update", "Metrics", ResourceToJSON(nodeMetrics))
			fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(nodeMetrics.Kind))
			events <- event
		}

		time.Sleep(15 * time.Second)
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	topicID := os.Getenv("KAFKA_TOPIC")
	kafka_url := os.Getenv("KAFKA_BROKER_URL")
	client := publisher.Producer(kafka_url)

	// client, err := publisher.ClientFromFile("config.json")

	// topicID := "0.0.1003"

	fmt.Printf("Publishing to topicID: %v\n", topicID)

	k8sclient := kubestate.K8sClientSet()

	coreV1Client := k8sclient.CoreV1()
	appsV1Client := k8sclient.AppsV1()
	restClient := k8sclient.RESTClient()

	namespace := "default"

	queue := make(chan Event, 1000)
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, topicID, queue, &wg)

	event := NewEvent("Add", "Cluster", nil)
	fmt.Printf("%s Sending Event: %s %s\n", event.Id, event.Action, event.Kind)
	queue <- event

	nodes, err := coreV1Client.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d nodes in the cluster\n", len(nodes.Items))

	for _, k8snode := range nodes.Items {
		event := NewEvent("Add", "Node", ResourceToJSON(k8snode))
		fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(k8snode.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	configmaps, err := coreV1Client.ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d configmaps in the cluster\n", len(configmaps.Items))

	for _, configmap := range configmaps.Items {
		event := NewEvent("Add", "ConfigMap", ResourceToJSON(configmap))
		fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(configmap.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	deployments, err := appsV1Client.Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d deployments in the cluster\n", len(deployments.Items))

	for _, deployment := range deployments.Items {
		event := NewEvent("Add", "Deployment", ResourceToJSON(deployment))
		fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(deployment.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	replicasets, err := appsV1Client.ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d replicasets in the cluster\n", len(replicasets.Items))

	for _, replicaset := range replicasets.Items {
		event := NewEvent("Add", "ReplicaSet", ResourceToJSON(replicaset))
		fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(replicaset.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	pods, err := coreV1Client.Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d pods in the cluster\n", len(pods.Items))

	for _, pod := range pods.Items {
		event := NewEvent("Add", "Pod", ResourceToJSON(pod))
		fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(pod.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	services, err := coreV1Client.Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d services in the cluster\n", len(services.Items))

	for _, service := range services.Items {
		event := NewEvent("Add", "Service", ResourceToJSON(service))
		fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(service.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	go metricsWorker(restClient, queue)

	watcher, err := k8sclient.CoreV1().Events(namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	currentTime := time.Now()

eventLoop:
	for event := range watcher.ResultChan() {
		item := event.Object.(*v1.Event)

		if item.CreationTimestamp.Time.Before(currentTime) {
			continue
		}

		switch event.Type {
		case watch.Added:
			if item.InvolvedObject.Kind == "Pod" {
				switch item.Reason {
				case "Killing":
					fmt.Printf("\nKubernetes API Server: Deleted Pod %s\n", item.InvolvedObject.Name)
					event := NewEvent("Delete", "Pod", ResourceToJSON(item.InvolvedObject.Name))
					fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- event
					// println(string(ToJSON(event)))
				case "Started":
					time.Sleep(3 * time.Second)
					fmt.Printf("\nKubernetes API Server: Started Pod %s\n", item.InvolvedObject.Name)
					if item.InvolvedObject.Name == "done" {
						event := NewEvent("Add", "Done", ResourceToJSON(""))
						fmt.Printf("%s Sending Event: %s %s\n", event.Id, event.Action, event.Kind)
						// Send(client, topicID, done, metadata)
						queue <- event
						fmt.Println("Received 'Done' signal, closing worker chan")
						close(queue)
						wg.Wait()
						break eventLoop
					}

					var pod *v1.Pod
					trys := 0

					for {
						pod, err = k8sclient.CoreV1().Pods(namespace).Get(context.TODO(), string(item.InvolvedObject.Name), metav1.GetOptions{})
						if err != nil {
							println(err.Error())
							trys += 1
							time.Sleep(1 * time.Second)
							if trys == 5 {
								break
							}
						} else {
							break
						}
					}

					node, err := k8sclient.CoreV1().Nodes().Get(context.TODO(), string(pod.Spec.NodeName), metav1.GetOptions{})
					if err != nil {
						println(err.Error())
						continue
					}
					images := node.Status.Images
					for _, image := range images {
						for _, name := range image.Names {
							if name == pod.Status.ContainerStatuses[0].ImageID {
								newImage := Image{
									NodeUID: string(node.GetUID()),
									Data:    image,
								}
								event := NewEvent("Add", "Image", ResourceToJSON(newImage))
								fmt.Printf("%s Sending Event: %s %s %s on K8sNode %s\n", event.Id, event.Action, event.Kind, string(image.Names[len(image.Names)-1]), string(pod.Spec.NodeName))
								// Send(client, topicID, event, metadata)
								queue <- event
								// println(string(ToJSON(event)))
							}
						}
					}

					event := NewEvent("Add", "Pod", ResourceToJSON(*pod))
					fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- event
					// println(string(ToJSON(event)))

					// fmt.Printf(" name: %s, imageID %s", pod.Spec.NodeName, pod.Status.ContainerStatuses[0].ImageID)
				}
			}

			if item.InvolvedObject.Kind == "Node" {
				switch item.Reason {
				case "RemovingNode":
					fmt.Printf("\nKubernetes API Server: Removing Node %s\n", item.InvolvedObject.Name)
					event := NewEvent("Delete", "Node", ResourceToJSON((item.InvolvedObject.Name)))
					fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- event
				case "Starting":
					fmt.Printf("\nKubernetes API Server: Starting Node %s\n", item.InvolvedObject.Name)
					node, err := k8sclient.CoreV1().Nodes().Get(context.TODO(), string(item.InvolvedObject.Name), metav1.GetOptions{})
					if err != nil {
						println(err.Error())
						continue
					}
					event := NewEvent("Add", "Node", ResourceToJSON(*node))
					fmt.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- event
				}
			}
		case watch.Modified:
		case watch.Deleted:
		}
	}
}
