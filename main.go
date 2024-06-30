package main

import (
	"encoding/json"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	Hostname  string `json:"hostname"`
	IpAddress string `json:"ipaddress"`

	Timestamp time.Time `json:"-"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Panicf("Error loading .env file : %s", err)
	}

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Panicf("Failed to connect to broker : %s", err)
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("Failed to open a channel : %s", err)
	}

	q, err := ch.QueueDeclare(
		"onboarding",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to declare a queue : %s", err)
	}

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
			msg := Message{Timestamp: d.Timestamp}
			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				log.Printf("Error unmarshalling message : %s", err)
			}

			//Make request to the rest of API
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
