package publisher

type Publisher interface {
	SubmitMessage(topicID string, message []byte) string
	NewTopic(topicID string)
}
