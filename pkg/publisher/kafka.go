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
	Topics      map[string]bool
}

func (p KafkaPublisher) SubmitMessage(topicID string, message []byte) string {

	if created, exists := p.Topics[topicID]; exists && created {

		delivery_chan := make(chan kafka.Event, 10000)
		err := p.Client.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topicID, Partition: kafka.PartitionAny},
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
	} else {
		return fmt.Sprintf("The topic %s does not exist", topicID)
	}
}

func (p *KafkaPublisher) NewTopic(topicID string) {

	topicSpecification := kafka.TopicSpecification{
		Topic:             topicID,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := p.AdminClient.CreateTopics(ctx, []kafka.TopicSpecification{topicSpecification})
	if err != nil {
		log.Fatalf("Failed to create topic: %s", err)
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			log.Fatalf("Failed to create topic %s: %v", result.Topic, result.Error)
		} else {
			p.Topics[result.Topic] = true
			fmt.Printf("Successfully created topic %s\n", result.Topic)
		}
	}
}

func (p KafkaPublisher) Ack() {
	for e := range p.Client.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				fmt.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
			} else {
				fmt.Printf("Successfully produced record to topic %s partition [%d] @ offset %v\n",
					*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
			}
		}
	}
}
