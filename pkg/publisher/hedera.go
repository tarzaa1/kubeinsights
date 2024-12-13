package publisher

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hashgraph/hedera-sdk-go/v2"
)

type HederaPublisher struct {
	Client *hedera.Client
}

func NewHederaPublisher(config HederaConfig) *HederaPublisher {
	client := HederaClient(config)
	return &HederaPublisher{Client: client}
}

func (p HederaPublisher) SubmitMessage(message []byte, topic string) string {

	hedera_topicID, err := hedera.TopicIDFromString(topic)
	if err != nil {
		panic(err.Error())
	}

	submitMessage, err := hedera.NewTopicMessageSubmitTransaction().
		SetMessage(message).
		SetTopicID(hedera_topicID).
		SetMaxChunks(40).
		Execute(p.Client)
	if err != nil {
		panic(err.Error())
	}

	receipt, err := submitMessage.GetReceipt(p.Client)

	if err != nil {
		return err.Error()
	}

	transactionStatus := receipt.Status
	return transactionStatus.String()
}

func NewHederaTopic(client *hedera.Client) {

	transactionResponse, err := hedera.NewTopicCreateTransaction().
		Execute(client)
	if err != nil {
		panic(err.Error())
	}

	transactionReceipt, err := transactionResponse.GetReceipt(client)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("New topicID: %v\n", transactionReceipt.TopicID)
}

func NewHederaAccount(Client *hedera.Client) (hedera.AccountID, hedera.PrivateKey) {

	privateKey, err := hedera.PrivateKeyGenerateEd25519()
	if err != nil {
		panic(err.Error())
	}

	publicKey := privateKey.PublicKey()

	fmt.Printf("Private key: %v\n", privateKey.String())
	fmt.Printf("Public key: %v\n", publicKey.String())

	newAccount, err := hedera.NewAccountCreateTransaction().
		SetKey(publicKey).
		SetInitialBalance(hedera.NewHbar(1000)).
		Execute(Client)

	if err != nil {
		panic(err.Error())
	}

	receipt, err := newAccount.GetReceipt(Client)
	if err != nil {
		panic(err.Error())
	}

	newAccountId := receipt.AccountID
	fmt.Printf("accountID: %v\n", newAccountId)

	return *newAccountId, privateKey
}

type HederaConfig struct {
	Network struct {
		AccountId string `json:"accountId"`
		Address   string `json:"address"`
	} `json:"network"`
	Operator struct {
		AccountId  string `json:"accountId"`
		PrivateKey string `json:"privateKey"`
	} `json:"operator"`
	MirrorNode string `json:"mirrorNetwork"`
}

func ReadHederaConfig(filepath string) HederaConfig {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		panic(err.Error())
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err.Error())
	}

	var config HederaConfig
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		panic(err.Error())
	}

	return config
}

func WriteHederaConfig(config HederaConfig, filePath string) {

	jsonData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		panic(err.Error())
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err.Error())
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		panic(err.Error())
	}
}

func CreateHederaClient(networkAddress string, networkAccountId string, operatorAccountId string, operatorPrivateKey string, mirrorAddress string) *hedera.Client {

	node := make(map[string]hedera.AccountID, 1)
	networkAccountID, err := hedera.AccountIDFromString(networkAccountId)
	if err != nil {
		panic(err.Error())
	}

	node[networkAddress] = networkAccountID

	mirrorNode := []string{mirrorAddress}

	client := hedera.ClientForNetwork(node)
	client.SetMirrorNetwork(mirrorNode)

	accountId, err := hedera.AccountIDFromString(operatorAccountId)
	if err != nil {
		panic(err.Error())
	}
	privateKey, err := hedera.PrivateKeyFromString(operatorPrivateKey)
	if err != nil {
		panic(err.Error())
	}

	client.SetOperator(accountId, privateKey)

	return client
}

func HederaClient(config HederaConfig) *hedera.Client {

	networkAddress := config.Network.Address
	networkAccountId := config.Network.AccountId
	mirrorAddress := config.MirrorNode
	operatorAccountId := config.Operator.AccountId
	operatorPrivateKey := config.Operator.PrivateKey

	client := CreateHederaClient(networkAddress, networkAccountId, operatorAccountId, operatorPrivateKey, mirrorAddress)

	return client
}
