package main

import (
	"log"
	"sync"

	"github.com/tarzaa1/kubeinsights/pkg/event"
	"github.com/tarzaa1/kubeinsights/pkg/publisher"
)

func worker(p publisher.Publisher, topic string, log *log.Logger, events <-chan event.Event, wg *sync.WaitGroup) {
	defer wg.Done()
	for e := range events {
		status := p.SubmitMessage(e.Bytes(), topic)
		log.Printf("%s %s", e.Id, status)
	}
}

func dualPublisherWorker(p1 publisher.Publisher, p1Topic string, p2 publisher.Publisher, p2Topic string, log *log.Logger, events <-chan event.Event, wg *sync.WaitGroup) {
	defer wg.Done()
	for e := range events {
		b := e.Bytes()
		status := p1.SubmitMessage(b, p1Topic)
		log.Printf("%s %s", e.Id, status)
		status = p2.SubmitMessage(b, p2Topic)
		log.Printf("%s %s", e.Id, status)
	}
}
