package publisher

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaPublisher struct {
	Client      *kafka.Producer
	AdminClient *kafka.AdminClient
}

func NewKafkaPublisher(address string) *KafkaPublisher {

	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": address})
	if err != nil {
		log.Fatalf("Failed to create Admin client: %s", err)
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": address})

	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}

	return &KafkaPublisher{Client: producer, AdminClient: adminClient}
}

func (p KafkaPublisher) SubmitMessage(message []byte, topic string) string {

	delivery_chan := make(chan kafka.Event, 10000)
	err := p.Client.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message},
		delivery_chan,
	)

	if err != nil {
		close(delivery_chan)
		return fmt.Sprintf("Produce failed: %s\n", err)
	}

	e := <-delivery_chan
	m := e.(*kafka.Message)

	var status string

	if m.TopicPartition.Error != nil {
		status = fmt.Sprintf("Delivery failed: %v\n", m.TopicPartition.Error)
	} else {
		status = fmt.Sprintf("Delivered message to topic %s [%d] at offset %v\n",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	}
	close(delivery_chan)
	return status
}

func (p KafkaPublisher) NewTopic(topicID string) (string, error) {

	topicSpecification := kafka.TopicSpecification{
		Topic:             topicID,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := p.AdminClient.CreateTopics(ctx, []kafka.TopicSpecification{topicSpecification})
	if err != nil {
		return "", fmt.Errorf("failed to create topic: %s", err)
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			if result.Error.Code() == kafka.ErrTopicAlreadyExists {
				log.Printf("topic `%s` already exists", topicID)
				return topicID, nil
			} else {
				return "", fmt.Errorf("failed to create topic %s: %v", result.Topic, result.Error)
			}
		} else {
			fmt.Printf("Successfully created topic %s\n", result.Topic)
			return topicID, nil
		}
	}

	return "", fmt.Errorf("failed to create Topic! Did not get back any results")
}
