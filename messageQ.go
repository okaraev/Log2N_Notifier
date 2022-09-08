package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/streadway/amqp"
)

func ReceiveMessage(connectionParams interface{}) (<-chan interface{}, error) {
	abstractchan := make(chan interface{})
	confParams, ok := connectionParams.(qconfig)
	if !ok {
		return nil, fmt.Errorf("connectionparams argument is not 'qconfig' type")
	}
	connectRabbitMQ, err := amqp.Dial(confParams.QConnectionString)
	if err != nil {
		return nil, err
	}
	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		connectRabbitMQ.Close()
		return nil, err
	}
	consumerName := os.Getenv("COMPUTERNAME")
	messages, err := channelRabbitMQ.Consume(confParams.QName, consumerName, false, false, false, false, nil)
	if err != nil {
		connectRabbitMQ.Close()
		channelRabbitMQ.Close()
		return nil, err
	}
	go func() {
		for mess := range messages {
			abstractchan <- mess
		}
	}()
	return abstractchan, nil
}

func DelayMessage(message interface{}, connectionParams interface{}) error {
	confParams, ok := connectionParams.(qconfig)
	if !ok {
		return fmt.Errorf("connectionparams argument is not 'qconfig' type")
	}
	notification, ok := message.(Notification)
	if !ok {
		return fmt.Errorf("message is not notification")
	}
	connectRabbitMQ, err := amqp.Dial(confParams.QConnectionString)
	if err != nil {
		return err
	}
	defer connectRabbitMQ.Close()
	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		return err
	}
	defer channelRabbitMQ.Close()
	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	HoldTime := notification.HoldTime
	DelayQ := fmt.Sprintf("DelayedQ_%d", HoldTime)
	HoldTimems := HoldTime * 1000
	_, err = channelRabbitMQ.QueueDeclare(
		DelayQ, // Q Name
		true,   // Durable
		false,  // auto delete
		false,  // exclusive
		false,  // no wait
		amqp.Table{
			"x-dead-letter-exchange":    "",               // Excchange where to send after TTL expire
			"x-dead-letter-routing-key": confParams.QName, // Queue name
			"x-message-ttl":             HoldTimems,       // Q Messages expiration time before moving to other Q
			"x-expires":                 HoldTimems * 2,   // Delete Q after x time
		}, // arguments
	)
	if err != nil {
		return err
	}
	data := amqp.Publishing{
		ContentType: "application/json",
		Body:        bytes,
	}
	err = channelRabbitMQ.Publish("", DelayQ, false, false, data)
	if err != nil {
		return err
	}
	return nil
}

func ProcessMessage(message interface{}, Delayer func(message interface{}) error) error {
	delivery, ok := message.(amqp.Delivery)
	if !ok {
		return fmt.Errorf("message argument is not delivery")
	}
	notification := Notification{}
	Note := GetNotificationOperation()
	err := json.Unmarshal(delivery.Body, &notification)
	if err != nil {
		return err
	}
	err = Note.Execute(notification)
	if err != nil {
		log.Println(err)
		errmessage := fmt.Sprint(err)
		if !strings.Contains(errmessage, "cannot find method") {
			if notification.Retried < notification.RetryCount {
				notification.Retried++
				err = Delayer(notification)
				if err != nil {
					log.Println(err)
				}
			} else {
				log.Printf("Skipping after %d retries\nLog: %s\nMethod: %s", notification.Retried, notification.Log, notification.NotificationMethod)
			}
		}
	}
	err = delivery.Ack(true)
	if err != nil {
		log.Println(err)
		return (fmt.Errorf("cannot acknowledge; %v", err))
	}
	return nil
}
