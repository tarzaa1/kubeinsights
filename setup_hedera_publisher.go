package main

import (
	"fmt"
	"kubeinsights/hedera"
)

func main() {
	var config hedera.Config
	config, err := hedera.GetConfig("default_config.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	networkAddress := config.Network.Address
	networkAccountId := config.Network.AccountId
	mirrorAddress := config.MirrorNode
	operatorAccountId := config.Operator.AccountId
	operatorPrivateKey := config.Operator.PrivateKey

	client, err := hedera.CreateClient(networkAddress, networkAccountId, operatorAccountId, operatorPrivateKey, mirrorAddress)

	if err != nil {
		panic(err.Error())
	}

	newAccountId, newPrivateKey, err := hedera.NewAccount(client)
	if err != nil {
		fmt.Println(err)
		return
	}

	config.Operator.AccountId = newAccountId.String()
	config.Operator.PrivateKey = newPrivateKey.String()

	hedera.WriteConfig(config, "config.json")

	client, err = hedera.ClientFromFile("config.json")

	if err != nil {
		panic(err.Error())
	}

	topicID, err := hedera.NewTopic(client)
	if err != nil {
		return
	}

	fmt.Printf("New topicID: %v\n", topicID)
}
