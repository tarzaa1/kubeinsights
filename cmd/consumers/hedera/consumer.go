package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	sub "kubeinsights/pkg/hedera"

	"github.com/hashgraph/hedera-sdk-go/v2"
)

func main() {

	client, err := sub.ClientFromFile("sub_config.json")

	if err != nil {
		fmt.Printf("Failed to create hedera client: %s\n", err)
		os.Exit(1)
	}

	// Get the topic ID from the transaction receipt
	topicID, err := hedera.TopicIDFromString("0.0.1003")

	if err != nil {
		println(err.Error(), ": error creating topic")
		return
	}

	//Log the topic ID to the console
	fmt.Printf("topicID: %v\n", topicID)

	// Subscribe to the topic
	_, err = hedera.NewTopicMessageQuery().
		SetTopicID(topicID).
		Subscribe(client.Client, func(message hedera.TopicMessage) {
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
