package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/tarzaa1/kubeinsights/pkg/kubestate"
	"github.com/tarzaa1/kubeinsights/pkg/publisher"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
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

func Send(p publisher.Publisher, event Event, topic string) string {
	return p.SubmitMessage(event.MarshalToJSON(), topic)
}

func worker(p publisher.Publisher, topic string, logger *log.Logger, events <-chan Event, wg *sync.WaitGroup) {
	defer wg.Done()
	for event := range events {
		status := Send(p, event, topic)
		logger.Printf("%s %s", event.Id, status)
	}
}

func khworker(p1 publisher.Publisher, p1_topic string, p2 publisher.Publisher, p2_topic string, logger *log.Logger, events <-chan Event, wg *sync.WaitGroup) {
	defer wg.Done()
	for event := range events {
		status1 := Send(p1, event, p1_topic)
		logger.Printf("%s %s", event.Id, status1)
		status2 := Send(p2, event, p2_topic)
		logger.Printf("%s %s", event.Id, status2)
	}
}

func metricsWorker(metricsClient *metrics.Clientset, loggers Loggers, events chan<- Event, namespace string) {
	// nodeMetricsList := metrics.NodeMetricsList{}
	for {
		nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			loggers.ErrorLogger.Println(err)
			loggers.ErrorLogger.Println("Please setup metrics server... Trying again in 10 seconds")
			time.Sleep(10 * time.Second)
			continue
		}

		nodeMetricsEvent := NewEvent("Update", "NodeMetrics", ResourceToJSON(nodeMetrics))
		loggers.InfoLogger.Printf("%s Sending Event: %s %s\n", nodeMetricsEvent.Id, nodeMetricsEvent.Action, nodeMetricsEvent.Kind)
		events <- nodeMetricsEvent

		// nodeprettyJSON, err := json.MarshalIndent(nodeMetrics, "", "    ")
		// if err != nil {
		// 	loggers.ErrorLogger.Println("Failed to pretty print metrics JSON:", err)
		// }

		// fmt.Println(string(nodeprettyJSON))

		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			loggers.ErrorLogger.Println("Error fetching Pod metrics:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		podMetricsEvent := NewEvent("Update", "PodMetrics", ResourceToJSON(podMetrics))
		loggers.InfoLogger.Printf("%s Sending Event: %s %s\n", podMetricsEvent.Id, podMetricsEvent.Action, podMetricsEvent.Kind)
		events <- podMetricsEvent

		// podprettyJSON, err := json.MarshalIndent(podMetrics, "", "    ")
		// if err != nil {
		// 	loggers.ErrorLogger.Println("Failed to pretty print metrics JSON:", err)
		// }

		// fmt.Println(string(podprettyJSON))

		time.Sleep(2 * time.Second)
	}
}

type Loggers struct {
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
}

func main() {

	errorLogger := log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	loggers := Loggers{
		InfoLogger:  infoLogger,
		ErrorLogger: errorLogger,
	}

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	config, clientset := kubestate.K8sClientSet()

	coreV1Client := clientset.CoreV1()
	appsV1Client := clientset.AppsV1()
	// restClient := clientset.RESTClient()
	networkClient := clientset.NetworkingV1()

	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	namespace := os.Getenv("NAMESPACE")

	dest := os.Getenv("DATA_DEST")
	queue := make(chan Event, 1000)
	var wg sync.WaitGroup
	wg.Add(1)

	var p publisher.Publisher
	var topicID string

	if dest == "kafka" {
		topicID = os.Getenv("KAFKA_TOPIC")
		kafka_url := os.Getenv("KAFKA_BROKER_URL")
		p = publisher.NewKafkaPublisher(kafka_url)
		// fmt.Println(p.SubmitMessage([]byte(topicID), "my-topic"))

		infoLogger.Printf("Publishing to Kafka topicID: %v\n", topicID)
		go worker(p, topicID, infoLogger, queue, &wg)

	} else if dest == "hedera" {
		topicID = "0.0.1003"
		config := publisher.ReadHederaConfig("config.json")
		p = publisher.NewHederaPublisher(config)

		infoLogger.Printf("Publishing to Hedera topicID: %v\n", topicID)
		go worker(p, topicID, infoLogger, queue, &wg)

	} else {
		hedera_topicID := "0.0.1003"
		config := publisher.ReadHederaConfig("config.json")
		p1 := publisher.NewHederaPublisher(config)
		new_hedera_topicID, err := p1.NewTopic("memo")
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("New topicID: %s\n", new_hedera_topicID)
		message := map[string]string{"topic": new_hedera_topicID, "cluster": os.Getenv("KAFKA_TOPIC")}
		msg, err := json.Marshal(message)
		if err != nil {
			panic(err.Error())
		}

		p1.SubmitMessage(msg, hedera_topicID)

		time.Sleep(10 * time.Second)

		kafka_topicID := os.Getenv("KAFKA_TOPIC")
		kafka_url := os.Getenv("KAFKA_BROKER_URL")
		p2 := publisher.NewKafkaPublisher(kafka_url)

		infoLogger.Printf("Publishing to Kafka topicID %q and Hedera topicID %q\n", kafka_topicID, new_hedera_topicID)
		go khworker(p1, new_hedera_topicID, p2, kafka_topicID, infoLogger, queue, &wg)
	}

	event := NewEvent("Add", "Cluster", nil)
	infoLogger.Printf("%s Sending Event: %s %s\n", event.Id, event.Action, event.Kind)
	queue <- event

	nodes, err := coreV1Client.Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	infoLogger.Printf("Kubernetes API Server: There are %d nodes in the cluster\n", len(nodes.Items))

	for _, k8snode := range nodes.Items {
		event := NewEvent("Add", "Node", ResourceToJSON(k8snode))
		infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(k8snode.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	configmaps, err := coreV1Client.ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	infoLogger.Printf("Kubernetes API Server: There are %d configmaps in the cluster\n", len(configmaps.Items))

	for _, configmap := range configmaps.Items {
		event := NewEvent("Add", "ConfigMap", ResourceToJSON(configmap))
		infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(configmap.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	deployments, err := appsV1Client.Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	infoLogger.Printf("Kubernetes API Server: There are %d deployments in the cluster\n", len(deployments.Items))

	for _, deployment := range deployments.Items {
		event := NewEvent("Add", "Deployment", ResourceToJSON(deployment))
		infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(deployment.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	replicasets, err := appsV1Client.ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	infoLogger.Printf("Kubernetes API Server: There are %d replicasets in the cluster\n", len(replicasets.Items))

	for _, replicaset := range replicasets.Items {
		event := NewEvent("Add", "ReplicaSet", ResourceToJSON(replicaset))
		infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(replicaset.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	pods, err := coreV1Client.Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	infoLogger.Printf("Kubernetes API Server: There are %d pods in the cluster\n", len(pods.Items))

	for _, pod := range pods.Items {
		event := NewEvent("Add", "Pod", ResourceToJSON(pod))
		infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(pod.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	services, err := coreV1Client.Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	infoLogger.Printf("Kubernetes API Server: There are %d services in the cluster\n", len(services.Items))

	for _, service := range services.Items {
		event := NewEvent("Add", "Service", ResourceToJSON(service))
		infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(service.Name))
		// Send(client, topicID, event, metadata)
		queue <- event
	}

	ingresses, err := networkClient.Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	infoLogger.Printf("Kubernetes API Server: There are %d ingresses in the cluster\n", len(ingresses.Items))

	for _, ingress := range ingresses.Items {
		event := NewEvent("Add", "Ingress", ResourceToJSON(ingress))
		infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(ingress.Name))
		queue <- event
	}

	go metricsWorker(metricsClient, loggers, queue, namespace)

	var resourceVersion string

	initialList, err := coreV1Client.Events(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		loggers.ErrorLogger.Printf("Error listing events: %v\n", err)
		time.Sleep(5 * time.Second)
		return
	}
	resourceVersion = initialList.ResourceVersion

	// not processing the initial list as i already handled them earlier

	for {
		watcher, err := coreV1Client.Events(namespace).Watch(context.TODO(), metav1.ListOptions{
			ResourceVersion: resourceVersion,
			Watch:           true,
		})
		if err != nil {
			loggers.ErrorLogger.Printf("Error creating watcher: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		ch := watcher.ResultChan()
	watchLoop:
		for event := range ch {
			switch event.Type {
			case watch.Added:
				k8sEvent, ok := event.Object.(*v1.Event)
				if !ok {
					continue
				}

				resourceVersion = k8sEvent.ResourceVersion

				if k8sEvent.InvolvedObject.Kind == "Pod" {
					switch k8sEvent.Reason {
					case "Killing":
						infoLogger.Printf("\nKubernetes API Server: Deleted Pod %s\n", k8sEvent.InvolvedObject.Name)
						event := NewEvent("Delete", "Pod", ResourceToJSON(k8sEvent.InvolvedObject.Name))
						infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(k8sEvent.InvolvedObject.Name))
						queue <- event
					case "Started":
						time.Sleep(3 * time.Second)
						infoLogger.Printf("\nKubernetes API Server: Started Pod %s\n", k8sEvent.InvolvedObject.Name)
						if k8sEvent.InvolvedObject.Name == "done" {
							event := NewEvent("Add", "Done", ResourceToJSON(""))
							infoLogger.Printf("%s Sending Event: %s %s\n", event.Id, event.Action, event.Kind)
							queue <- event
							fmt.Println("Received 'Done' signal, closing worker chan")
							close(queue)
							wg.Wait()
							break watchLoop
						}

						var pod *v1.Pod
						trys := 0

						for {
							pod, err = coreV1Client.Pods(namespace).Get(context.TODO(), string(k8sEvent.InvolvedObject.Name), metav1.GetOptions{})
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

						node, err := coreV1Client.Nodes().Get(context.TODO(), string(pod.Spec.NodeName), metav1.GetOptions{})
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
									infoLogger.Printf("%s Sending Event: %s %s %s on K8sNode %s\n", event.Id, event.Action, event.Kind, string(image.Names[len(image.Names)-1]), string(pod.Spec.NodeName))
									queue <- event
								}
							}
						}

						event := NewEvent("Add", "Pod", ResourceToJSON(*pod))
						infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(k8sEvent.InvolvedObject.Name))
						queue <- event
					}
				}

				if k8sEvent.InvolvedObject.Kind == "Node" {
					switch k8sEvent.Reason {
					case "RemovingNode":
						infoLogger.Printf("\nKubernetes API Server: Removing Node %s\n", k8sEvent.InvolvedObject.Name)
						event := NewEvent("Delete", "Node", ResourceToJSON((k8sEvent.InvolvedObject.Name)))
						infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(k8sEvent.InvolvedObject.Name))
						queue <- event
					case "Starting":
						infoLogger.Printf("\nKubernetes API Server: Starting Node %s\n", k8sEvent.InvolvedObject.Name)
						node, err := coreV1Client.Nodes().Get(context.TODO(), string(k8sEvent.InvolvedObject.Name), metav1.GetOptions{})
						if err != nil {
							println(err.Error())
							continue
						}
						event := NewEvent("Add", "Node", ResourceToJSON(*node))
						infoLogger.Printf("%s Sending Event: %s %s %s\n", event.Id, event.Action, event.Kind, string(k8sEvent.InvolvedObject.Name))
						// Send(client, topicID, event, metadata)
						queue <- event
					}
				}

			case watch.Modified:
			case watch.Deleted:
			case watch.Error:
				statusErr, _ := event.Object.(*metav1.Status)
				loggers.ErrorLogger.Printf("Watch error received: %+v\n", statusErr)
				break watchLoop
			}
		}

		loggers.InfoLogger.Println("Watch channel closed (or broke out watch.Error), re-establishing...")
		list, err := coreV1Client.Events(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			loggers.ErrorLogger.Printf("Error listing events after watch closed: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		resourceVersion = list.ResourceVersion
	}
}
