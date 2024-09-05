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
	"strconv"
	"time"
)

func failWithError(err error, formatString string, args ...any) {
	if err != nil {
		util.Err(fmt.Errorf(fmt.Sprintf("%s: %w",formatString), append(args, err)...).Error())
	}
}

var RETRY_DELAY = 5

func main() {
	err := godotenv.Load()
	failWithError(err, "Error loading .env file")

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	failWithError(err, "Failed to connect to broker")

	defer conn.Close()

	ch, err := conn.Channel()
	failWithError(err, "Failed to open a channel")

	incomingQueue := os.Getenv("RABBITMQ_INCOMING_QUEUE")
	outgoingQueue := os.Getenv("RABBITMQ_OUTGOING_QUEUE")

	if value, ok := os.LookupEnv("RABBITMQ_RETRY_DELAY"); ok {
		if i, err := strconv.Atoi(value); err == nil {
			RETRY_DELAY = i
		}
	}

	inQ, err := ch.QueueDeclare(
		incomingQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	failWithError(err, "Failed to declare queue %s", incomingQueue)

	outQ, err := ch.QueueDeclare(
		outgoingQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	failWithError(err, "Failed to declare queue %s", outgoingQueue)

	exchangeArgs := map[string]interface{}{
		"x-delayed-type": "direct",
	}

	err = ch.ExchangeDeclare(
		incomingQueue,
		"x-delayed-message",
		true,
		false,
		false,
		false,
		exchangeArgs,
	)
	failWithError(err, "Failed to declare exchange %s", incomingQueue)

	err = ch.QueueBind(
		incomingQueue, // queue name
		incomingQueue, // routing key
		incomingQueue, // exchange
		false,
		nil)
	failWithError(err, "Failed to bind queue %s to exchange %s", incomingQueue, incomingQueue)

	// Consommation des messages
	msgs, err := ch.Consume(
		inQ.Name,   // nom de la queue
		"consumer", // consumer
		true,       // autoAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	failWithError(err, "Failed to register %s consumer", inQ.Name)
	util.Info("Connected to message broker")

	netbox := model.NewNetbox()
	err = netbox.Connect()
	failWithError(err, "Failed to connect to netbox")

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
					util.Warn("Error unmarshalling message : %w", err)
					return
				}

				//Make request to the rest of API
				err = netbox.CreateOrUpdateVM(msg)
				if err != nil {
					util.Warn("error creating or updating VM : %w", err)

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
						"x-delay": RETRY_DELAY * 1000,
					}

					chErr := ch.PublishWithContext(
						ctx,
						incomingQueue,
						inQ.Name,
						false,
						false,
						amqp.Publishing{
							ContentType: "application/json",
							Body:        newMsgJson,
							Headers:     headers,
						})

					if chErr != nil {
						util.Warn("Error re-publishing message: %s", chErr)
					} else {
						util.Warn("Re-sent message to RabbitMQ®️: %s", newMsgJson)
					}

					return
				}

				util.Success("VM %s is up to date", msg.Hostname)

				dur, _ := time.ParseDuration("10s")
				ctx, cancel := context.WithTimeout(context.Background(), dur)
				defer cancel()

				newMsg := msg

				newMsgJson, _ := json.Marshal(newMsg)

				chErr := ch.PublishWithContext(
					ctx,
					"",
					outQ.Name,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        newMsgJson,
					})

				if chErr != nil {
					util.Warn("Error publishing success message: %s", chErr)
				} else {
					util.Success("sent success message to RabbitMQ®️: %s", newMsgJson)
				}
			}()
		}
	}()

	util.Info(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
