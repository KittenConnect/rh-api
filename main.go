package main

import (
	"encoding/json"
	"fmt"
	"github.com/KittenConnect/rh-api/util"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
)

func failOnError(err error, msg string) {
	if err != nil {
		util.Err(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	err := godotenv.Load()
	failOnError(err, fmt.Sprintf("Error loading .env file : %s", err))

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	failOnError(err, fmt.Sprintf("Failed to connect to broker : %s", err))

	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, fmt.Sprintf("Failed to open a channel : %s", err))

	q, err := ch.QueueDeclare(
		"onboarding",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, fmt.Sprintf("Failed to declare a queue : %s", err))

	// Consommation des messages
	msgs, err := ch.Consume(
		q.Name,     // nom de la queue
		"consumer", // consumer
		true,       // autoAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	failOnError(err, "Failed to register a consumer")

	// Canal pour signaler la fin du programme
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			msg := model.Message{Timestamp: d.Timestamp}
			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				util.Warn(fmt.Sprintf("Error unmarshalling message : %s", err))
				continue
			}

			//Make request to the rest of API
		}
	}()

	util.Info(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
