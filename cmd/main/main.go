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
	"github.com/tarzaa1/kubeinsights/pkg/event"
	"github.com/tarzaa1/kubeinsights/pkg/kubestate"
	"github.com/tarzaa1/kubeinsights/pkg/publisher"
	"github.com/tarzaa1/kubeinsights/pkg/watcher"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	infoLog := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errLog := log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	if err := godotenv.Load(); err != nil {
		infoLog.Println("no .env file found")
	}

	config, clientset := kubestate.K8sClientSet()

	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		errLog.Fatalf("metrics client: %v", err)
	}

	namespace := os.Getenv("NAMESPACE")
	queue := make(chan event.Event, 1000)

	// publisher worker — drains the queue and submits to the configured destination
	var publisherWg sync.WaitGroup
	publisherWg.Add(1)
	setupPublisher(infoLog, errLog, queue, &publisherWg)

	ctx, cancel := context.WithCancel(context.Background())

	w := watcher.New(
		clientset.CoreV1(),
		clientset.AppsV1(),
		clientset.NetworkingV1(),
		metricsClient,
		queue,
		namespace,
		infoLog,
		errLog,
	)

	if err := w.Snapshot(ctx); err != nil {
		errLog.Fatalf("snapshot: %v", err)
	}

	var metricsWg sync.WaitGroup
	metricsWg.Add(1)
	go func() {
		defer metricsWg.Done()
		w.PollMetrics(ctx)
	}()

	w.Watch(ctx)   // blocks until "done" sentinel or context cancellation
	cancel()       // stop PollMetrics
	metricsWg.Wait()
	close(queue)   // signal publisher worker to drain and exit
	publisherWg.Wait()
}

func setupPublisher(infoLog, errLog *log.Logger, queue chan event.Event, wg *sync.WaitGroup) {
	dest := os.Getenv("DATA_DEST")
	switch dest {
	case "kafka":
		topic := os.Getenv("KAFKA_TOPIC")
		p := publisher.NewKafkaPublisher(os.Getenv("KAFKA_BROKER_URL"))
		infoLog.Printf("publishing to Kafka topic %q", topic)
		go worker(p, topic, infoLog, queue, wg)

	case "hedera":
		topic := "0.0.1003"
		cfg := publisher.ReadHederaConfig("config.json")
		p := publisher.NewHederaPublisher(cfg)
		infoLog.Printf("publishing to Hedera topic %q", topic)
		go worker(p, topic, infoLog, queue, wg)

	default:
		hederaCfg := publisher.ReadHederaConfig("config.json")
		p1 := publisher.NewHederaPublisher(hederaCfg)
		newTopic, err := p1.NewTopic("memo")
		if err != nil {
			errLog.Fatalf("creating Hedera topic: %v", err)
		}
		fmt.Printf("new Hedera topic: %s\n", newTopic)

		msg, err := json.Marshal(map[string]string{
			"topic":   newTopic,
			"cluster": os.Getenv("KAFKA_TOPIC"),
		})
		if err != nil {
			errLog.Fatalf("marshalling topic announcement: %v", err)
		}
		p1.SubmitMessage(msg, "0.0.1003")
		time.Sleep(10 * time.Second)

		kafkaTopic := os.Getenv("KAFKA_TOPIC")
		p2 := publisher.NewKafkaPublisher(os.Getenv("KAFKA_BROKER_URL"))
		infoLog.Printf("publishing to Kafka topic %q and Hedera topic %q", kafkaTopic, newTopic)
		go dualPublisherWorker(p1, newTopic, p2, kafkaTopic, infoLog, queue, wg)
	}
}
