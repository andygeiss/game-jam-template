package main

import (
	"fmt"

	"github.com/andygeiss/cloud-native-utils/messaging"
	"github.com/andygeiss/cloud-native-utils/service"
)

func main() {
	// Create a new context for a service and
	// exit on signals from the terminal.
	doneChan := make(chan bool)
	ctx, _ := service.Context()
	service.RegisterOnContextDone(ctx, func() {
		doneChan <- true
	})

	fmt.Println("🚀 consumer started")

	// Create a new Kafka dispatcher and subscribe to the topic.
	client := messaging.NewExternalDispatcher()

	// Create a new service function.
	fn := func(msg messaging.Message) (state messaging.MessageState, err error) {
		fmt.Printf("  📨 template - %s\n", string(msg.Data))
		doneChan <- true
		return messaging.MessageStateCompleted, nil
	}

	// Consume a message to the topic via the service function.
	client.Subscribe(ctx, "template", service.Wrap(fn))

	// Produce a message to the topic.
	_ = client.Publish(ctx, messaging.NewMessage(
		"template",
		[]byte("42!"),
	))

	// Wait for the context to be done.
	<-doneChan
	fmt.Println("\n👋 consumer stopped")
}
