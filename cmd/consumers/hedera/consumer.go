package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/tarzaa1/kubeinsights/pkg/publisher"

	"github.com/hashgraph/hedera-sdk-go/v2"
)

func main() {

	config := publisher.ReadHederaConfig("sub_config.json")
	client := publisher.HederaClient(config)

	topicID, err := hedera.TopicIDFromString("0.0.1003")

	if err != nil {
		log.Fatalf(err.Error(), ": error creating topic")
	}

	fmt.Printf("topicID: %v\n", topicID)

	// Subscribe to the topic
	_, err = hedera.NewTopicMessageQuery().
		SetTopicID(topicID).
		Subscribe(client, func(message hedera.TopicMessage) {
			var record map[string]interface{}
			err := json.Unmarshal(message.Contents, &record)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(message.ConsensusTimestamp.String())
		})

	if err != nil {
		println(err.Error(), ": error subscribing to topic")
		return
	}

	// Prevent the program from exiting to display the message from the mirror node to the console
	for {
		time.Sleep(30 * time.Second)
	}

}
