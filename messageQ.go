package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/streadway/amqp"
)

type ConnectionStatus struct {
	params      qconfig
	isConnected bool
}

type ConnectionList struct {
	List []ConnectionStatus
}

func (c *ConnectionList) GetConfig(action int) qconfig {
	i := 0
	for index, item := range c.List {
		if action == 1 {
			if !item.isConnected {
				i = index
			}
		} else {
			if item.isConnected {
				i = index
			}
		}
	}
	c.List[i].isConnected = true
	return c.List[i].params
}

func (c *ConnectionList) Release(conf qconfig) {
	for index, item := range c.List {
		if item.params.QConnectionString == conf.QConnectionString && item.params.QName == conf.QName {
			c.List[index].isConnected = false
		}
	}
}

func ReceiveMessage() (<-chan interface{}, error) {
	abstractchan := make(chan interface{})
	confParams := ConnList.GetConfig(1)
	connectRabbitMQ, err := amqp.Dial(confParams.QConnectionString)
	if err != nil {
		return nil, err
	}
	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		ConnList.Release(confParams)
		connectRabbitMQ.Close()
		return nil, err
	}
	consumerName := os.Getenv("COMPUTERNAME")
	messages, err := channelRabbitMQ.Consume(confParams.QName, consumerName, false, false, false, false, nil)
	if err != nil {
		ConnList.Release(confParams)
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

func DelayMessage(message interface{}) error {
	notification, ok := message.(Notification)
	if !ok {
		return fmt.Errorf("message is not notification")
	}
	confParams := ConnList.GetConfig(2)
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
