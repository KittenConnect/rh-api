package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/KittenConnect/rh-api/model"
	"github.com/KittenConnect/rh-api/util"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"time"
)

func failOnError(err error, msg string) {
	if err != nil {
		util.Err(fmt.Errorf("%s: %w", msg, err).Error())
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

	incomingQueue := os.Getenv("RABBITMQ_INCOMING_QUEUE")
	outcomingQueue := os.Getenv("RABBITMQ_OUTGOING_QUEUE")

	q, err := ch.QueueDeclare(
		incomingQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, fmt.Sprintf("Failed to declare a queue : %s", err))

	err = ch.ExchangeDeclare(
		incomingQueue,
		"x-delayed-message",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, fmt.Sprintf("Failed to declare an exchange : %s", err))

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
	util.Info("Connected to message broker")

	netbox := model.NewNetbox()
	err = netbox.Connect()
	failOnError(err, "Failed to connect to netbox")

	if netbox.IsConnected() == false {
		util.Err("Unable to connect to netbox")
		os.Exit(-1)
	}

	// Canal pour signaler la fin du programme
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			go func() {
				msg := model.Message{Timestamp: d.Timestamp, FailCount: 20}
				err := json.Unmarshal(d.Body, &msg)
				if err != nil {
					util.Warn(fmt.Sprintf("Error unmarshalling message : %s", err))
					return
				}

				//Make request to the rest of API
				err = netbox.CreateOrUpdateVM(msg)
				if err != nil {
					util.Warn(fmt.Errorf("error creating or updating VM : %w", err).Error())

					dur, _ := time.ParseDuration("10s")
					ctx, cancel := context.WithTimeout(context.Background(), dur)
					defer cancel()

					newMsg := msg
					newMsg.FailCount--

					if newMsg.FailCount <= 0 {
						return
					}

					newMsgJson, _ := json.Marshal(newMsg)

					headers := amqp.Table{
						"x-delay": 60000,
					}

					chErr := ch.PublishWithContext(
						ctx,
						incomingQueue,
						q.Name,
						false,
						false,
						amqp.Publishing{
							ContentType: "application/json",
							Body:        newMsgJson,
							Headers:     headers,
						})

					if chErr != nil {
						util.Warn(fmt.Sprintf("Error re-publishing message : %s", chErr))
					} else {
						util.Warn(fmt.Sprintf("Re-sent message to RabbitMQ ®️ : %s", newMsgJson))
					}

					return
				}

				util.Success(fmt.Sprintf("VM up to date %s", msg.Hostname))

				dur, _ := time.ParseDuration("10s")
				ctx, cancel := context.WithTimeout(context.Background(), dur)
				defer cancel()

				newMsg := msg

				newMsgJson, _ := json.Marshal(newMsg)

				chErr := ch.PublishWithContext(
					ctx,
					"",
					outcomingQueue,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        newMsgJson,
					})

				if chErr != nil {
					util.Warn(fmt.Sprintf("Error publishing success message : %s", chErr))
				} else {
					util.Warn(fmt.Sprintf("sent success message to RabbitMQ ®️ : %s", newMsgJson))
				}
			}()
		}
	}()

	util.Info(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
