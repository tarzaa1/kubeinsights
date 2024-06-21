package main

import (
	"context"
	"encoding/csv"
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
)

type Event struct {
	Id        uuid.UUID       `json:"id"`
	Timestamp string          `json:"timestamp"`
	Action    string          `json:"action"`
	Kind      string          `json:"kind"`
	Body      json.RawMessage `json:"body"`
}

type Image struct {
	NodeUID string            `json:"nodeUID"`
	Data    v1.ContainerImage `json:"data"`
}

type EventLatency struct {
	ID      uuid.UUID
	Latency int64
}

var durations []EventLatency

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

func ResourceToJSON[T any](resource T) json.RawMessage {
	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		panic(err.Error())
	}
	return jsonBytes
}

func Send(client publisher.Client, topicID string, event Event, metadata string) {
	t1 := time.Now()
	event.SetTimestamp(t1.Format(time.RFC3339Nano))
	publisher.SubmitMessage(client, topicID, event.MarshalToJSON(), metadata)
	eventLatency := EventLatency{
		ID:      event.Id,
		Latency: time.Now().Sub(t1).Milliseconds(),
	}
	durations = append(durations, eventLatency)
}

type EventMetadata struct {
	event    Event
	metadata string
}

func worker(client publisher.Client, topicID string, queue chan EventMetadata, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		eventMetadata := <-queue
		event := eventMetadata.event
		metadata := eventMetadata.metadata

		if event.Kind == "Done" {
			Send(client, topicID, event, metadata)
			fmt.Println("Received 'Done' signal, worker stopping")
			return
		}

		Send(client, topicID, event, metadata)
	}
}

func writeDurationsToCSV(tuples []EventLatency, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write([]string{"ID", "Latency"})
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, t := range tuples {
		record := []string{t.ID.String(), fmt.Sprintf("%d", t.Latency)}
		err := writer.Write(record)
		if err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
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

	namespace := "cikm"

	queue := make(chan EventMetadata, 1000)

	var wg sync.WaitGroup

	wg.Add(1)

	go worker(client, topicID, queue, &wg)

	nodes, err := k8sclient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d nodes in the cluster\n", len(nodes.Items))

	for _, k8snode := range nodes.Items {
		event := NewEvent("Add", "Node", ResourceToJSON(k8snode))
		metadata := fmt.Sprintf("Hedera Message: Add K8sNode %s.", string(k8snode.Name))
		// Send(client, topicID, event, metadata)
		queue <- EventMetadata{event: event, metadata: metadata}
	}

	configmaps, err := k8sclient.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d configmaps in the cluster\n", len(configmaps.Items))

	for _, configmap := range configmaps.Items {
		event := NewEvent("Add", "ConfigMap", ResourceToJSON(configmap))
		metadata := fmt.Sprintf("Hedera Message: Add ConfigMap %s.", string(configmap.Name))
		// Send(client, topicID, event, metadata)
		queue <- EventMetadata{event: event, metadata: metadata}
	}

	deployments, err := k8sclient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d deployments in the cluster\n", len(deployments.Items))

	for _, deployment := range deployments.Items {
		event := NewEvent("Add", "Deployment", ResourceToJSON(deployment))
		metadata := fmt.Sprintf("Hedera Message: Add Deployment %s.", string(deployment.Name))
		// Send(client, topicID, event, metadata)
		queue <- EventMetadata{event: event, metadata: metadata}
	}

	replicasets, err := k8sclient.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d replicasets in the cluster\n", len(replicasets.Items))

	for _, replicaset := range replicasets.Items {
		event := NewEvent("Add", "ReplicaSet", ResourceToJSON(replicaset))
		metadata := fmt.Sprintf("Hedera Message: Add ReplicaSet %s.", string(replicaset.Name))
		// Send(client, topicID, event, metadata)
		queue <- EventMetadata{event: event, metadata: metadata}
	}

	pods, err := k8sclient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d pods in the cluster\n", len(pods.Items))

	for _, pod := range pods.Items {
		event := NewEvent("Add", "Pod", ResourceToJSON(pod))
		metadata := fmt.Sprintf("Hedera Message: Add Pod %s.", string(pod.Name))
		// Send(client, topicID, event, metadata)
		queue <- EventMetadata{event: event, metadata: metadata}
	}

	services, err := k8sclient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("\nKubernetes API Server: There are %d services in the cluster\n", len(services.Items))

	for _, service := range services.Items {
		event := NewEvent("Add", "Service", ResourceToJSON(service))
		metadata := fmt.Sprintf("Hedera Message: Add Service %s.", string(service.Name))
		// Send(client, topicID, event, metadata)
		queue <- EventMetadata{event: event, metadata: metadata}
	}

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
					metadata := fmt.Sprintf("Hedera Message: Delete Pod %s.", string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- EventMetadata{event: event, metadata: metadata}
					// println(string(ToJSON(event)))
				case "Started":
					time.Sleep(3 * time.Second)
					fmt.Printf("\nKubernetes API Server: Started Pod %s\n", item.InvolvedObject.Name)
					if item.InvolvedObject.Name == "done" {
						done := NewEvent("Add", "Done", ResourceToJSON(""))
						metadata := "Hedera Message: Done"
						// Send(client, topicID, done, metadata)
						queue <- EventMetadata{event: done, metadata: metadata}
						close(queue)
						wg.Wait()
						writeDurationsToCSV(durations, "durations.csv")
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
								metadata := fmt.Sprintf("Hedera Message: Add Image %s on K8sNode %s.", string(image.Names[len(image.Names)-1]), string(pod.Spec.NodeName))
								// Send(client, topicID, event, metadata)
								queue <- EventMetadata{event: event, metadata: metadata}
								// println(string(ToJSON(event)))
							}
						}
					}

					event := NewEvent("Add", "Pod", ResourceToJSON(*pod))
					metadata := fmt.Sprintf("Hedera Message: Add Pod %s.", string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- EventMetadata{event: event, metadata: metadata}
					// println(string(ToJSON(event)))

					// fmt.Printf(" name: %s, imageID %s", pod.Spec.NodeName, pod.Status.ContainerStatuses[0].ImageID)
				}
			}

			if item.InvolvedObject.Kind == "Node" {
				switch item.Reason {
				case "RemovingNode":
					fmt.Printf("\nKubernetes API Server: Removing Node %s\n", item.InvolvedObject.Name)
					event := NewEvent("Delete", "Node", ResourceToJSON((item.InvolvedObject.Name)))
					metadata := fmt.Sprintf("Hedera Message: Delete K8sNode %s.", string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- EventMetadata{event: event, metadata: metadata}
				case "Starting":
					fmt.Printf("\nKubernetes API Server: Starting Node %s\n", item.InvolvedObject.Name)
					node, err := k8sclient.CoreV1().Nodes().Get(context.TODO(), string(item.InvolvedObject.Name), metav1.GetOptions{})
					if err != nil {
						println(err.Error())
						continue
					}
					event := NewEvent("Add", "Node", ResourceToJSON(*node))
					metadata := fmt.Sprintf("Hedera Message: Add K8sNode %s.", string(item.InvolvedObject.Name))
					// Send(client, topicID, event, metadata)
					queue <- EventMetadata{event: event, metadata: metadata}
				}
			}
		case watch.Modified:
		case watch.Deleted:
		}
	}
}
