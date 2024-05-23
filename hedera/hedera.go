package hedera

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hashgraph/hedera-sdk-go/v2"
)

type Config struct {
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

type Client struct {
	*hedera.Client
}

func GetConfig(filepath string) (Config, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return Config{}, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return Config{}, err
	}

	var config Config

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func WriteConfig(config Config, filePath string) {

	jsonData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func NewAccount(client Client) (hedera.AccountID, hedera.PrivateKey, error) {

	privateKey, err := hedera.PrivateKeyGenerateEd25519()
	if err != nil {
		return hedera.AccountID{}, hedera.PrivateKey{}, err
	}

	publicKey := privateKey.PublicKey()

	fmt.Printf("Private key: %v\n", privateKey.String())
	fmt.Printf("Public key: %v\n", publicKey.String())

	newAccount, err := hedera.NewAccountCreateTransaction().
		SetKey(publicKey).
		SetInitialBalance(hedera.NewHbar(1000)).
		Execute(client.Client)

	if err != nil {
		println(err.Error(), ": error getting balance")
		return hedera.AccountID{}, hedera.PrivateKey{}, err
	}

	receipt, err := newAccount.GetReceipt(client.Client)
	if err != nil {
		println(err.Error(), ": error getting receipt")
		return hedera.AccountID{}, hedera.PrivateKey{}, err
	}

	newAccountId := receipt.AccountID
	fmt.Printf("accountID: %v\n", newAccountId)

	return *newAccountId, privateKey, nil
}

func CreateClient(networkAddress string, networkAccountId string, operatorAccountId string, operatorPrivateKey string, mirrorAddress string) (Client, error) {

	node := make(map[string]hedera.AccountID, 1)
	networkAccountID, err := hedera.AccountIDFromString(networkAccountId)
	if err != nil {
		return Client{}, err
	}

	node[networkAddress] = networkAccountID

	mirrorNode := []string{mirrorAddress}

	client := hedera.ClientForNetwork(node)
	client.SetMirrorNetwork(mirrorNode)

	accountId, err := hedera.AccountIDFromString(operatorAccountId)
	if err != nil {
		return Client{}, err
	}
	privateKey, err := hedera.PrivateKeyFromString(operatorPrivateKey)
	if err != nil {
		return Client{}, err
	}

	client.SetOperator(accountId, privateKey)

	return Client{client}, err
}

func ClientFromFile(filePath string) (Client, error) {
	config, err := GetConfig(filePath)
	if err != nil {
		fmt.Println(err)
		return Client{}, err
	}

	networkAddress := config.Network.Address
	networkAccountId := config.Network.AccountId
	mirrorAddress := config.MirrorNode
	operatorAccountId := config.Operator.AccountId
	operatorPrivateKey := config.Operator.PrivateKey

	client, err := CreateClient(networkAddress, networkAccountId, operatorAccountId, operatorPrivateKey, mirrorAddress)

	if err != nil {
		panic(err.Error())
	}

	return client, err
}

func NewTopic(client Client) (hedera.TopicID, error) {

	transactionResponse, err := hedera.NewTopicCreateTransaction().
		Execute(client.Client)
	if err != nil {
		println(err.Error(), ": error creating topic")
		return hedera.TopicID{}, err
	}

	transactionReceipt, err := transactionResponse.GetReceipt(client.Client)
	if err != nil {
		println(err.Error(), ": error getting topic create receipt")
		return hedera.TopicID{}, err
	}

	topicID := *transactionReceipt.TopicID

	// fmt.Printf("topicID: %v\n", topicID)

	return topicID, nil
}

func SubmitMessage(client Client, topicID string, message []byte, metadata string) {

	topic := TopicID(topicID)

	submitMessage, err := hedera.NewTopicMessageSubmitTransaction().
		SetMessage(message).
		SetTopicID(topic).
		SetMaxChunks(40).
		Execute(client.Client)
	if err != nil {
		println(err.Error(), ": error submitting to topic")
		return
	}

	receipt, err := submitMessage.GetReceipt(client.Client)

	if err != nil {
		println(err.Error())
		return
	}

	transactionStatus := receipt.Status
	fmt.Println(metadata + " " + transactionStatus.String())
}

func TopicID(topic string) hedera.TopicID {

	topicID, err := hedera.TopicIDFromString(topic)

	if err != nil {
		println(err.Error(), ": error getting topicID")
		return hedera.TopicID{}
	}

	// fmt.Printf("topicID: %v\n", topicID)

	return topicID
}
