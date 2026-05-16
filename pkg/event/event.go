package event

import (
	"encoding/json"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
)

type Event struct {
	Id        uuid.UUID       `json:"id"`
	Timestamp string          `json:"timestamp"`
	Action    string          `json:"action"`
	Kind      string          `json:"kind"`
	Body      json.RawMessage `json:"body"`
}

func New(action, kind string, body json.RawMessage) Event {
	return Event{
		Id:     uuid.New(),
		Action: action,
		Kind:   kind,
		Body:   body,
	}
}

func (e Event) Bytes() []byte {
	b, err := json.Marshal(e)
	if err != nil {
		panic(err.Error())
	}
	return b
}

// Image wraps a container image with the UID of the node that stores it.
type Image struct {
	NodeUID string            `json:"nodeUID"`
	Data    v1.ContainerImage `json:"data"`
}

// ToJSON serialises any value to a json.RawMessage. Panics on marshal error.
func ToJSON[T any](v T) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err.Error())
	}
	return b
}
