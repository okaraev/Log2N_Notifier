package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
)

type Retrier struct {
	Status      string        // Retrier Status
	HoldTime    time.Duration // Time duration after which Operation will retry. If Operation fails hold time multiplies until 360 seconds
	MaxHoldTime time.Duration
	Operation   func(q string, qq string) (<-chan amqp.Delivery, error)
}

func (r *Retrier) New(myFunc func(q string, qq string) (<-chan amqp.Delivery, error)) {
	r.HoldTime = 10 * time.Second
	r.MaxHoldTime = 360 * time.Second
	r.Status = "Closed"
	r.Operation = myFunc
}

func (r *Retrier) Open() {
	r.Status = "Open"
}

func (r *Retrier) Close() {
	r.HoldTime = 10 * time.Second
	r.Status = "Closed"
}

func (r *Retrier) Multiply() {
	if r.HoldTime <= r.MaxHoldTime {
		r.HoldTime = r.HoldTime * 2
	}
}

func (r *Retrier) isClosed() bool {
	return r.Status == "Closed"
}

func (r *Retrier) Do(RMQServer string, QName string) (<-chan amqp.Delivery, error) {
	result, err := r.Operation(RMQServer, QName)
	mydelivery := make(chan amqp.Delivery)
	if err != nil {
		r.Open()
		log.Println(err)
	}
	if r.isClosed() {
		// if status is success starting routine to receive messages
		go func() {
			for message := range result {
				mydelivery <- message
			}
			// Ending routine when error happened and setting status to fail
			r.Open()
		}()
	}
	// Starting forever loop in routine to check if status is fail
	go func() {
		for {
			time.Sleep(1 * time.Second)
			if !r.isClosed() {
				// if status is fail waiting for defined hold time and trying again
				time.Sleep(r.HoldTime)
				retryResult, err := r.Operation(RMQServer, QName)
				if err != nil {
					log.Println(err)
					r.Multiply()
				} else {
					// if retry is success setting status to success and starting routine to receive messages
					r.Close()
					go func() {
						for message := range retryResult {
							mydelivery <- message
						}
						// Ending routine when error happened and setting status to fail
						r.Open()
					}()
				}
			}
		}
	}()
	return (<-chan amqp.Delivery)(mydelivery), nil
}

func ReceiveMessage(RMQServer string, queue string) (<-chan amqp.Delivery, error) {
	connectRabbitMQ, err := amqp.Dial(RMQServer)
	if err != nil {
		return nil, err
	}
	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		connectRabbitMQ.Close()
		return nil, err
	}
	consumerName := os.Getenv("COMPUTERNAME")
	messages, err := channelRabbitMQ.Consume(queue, consumerName, false, false, false, false, nil)
	if err != nil {
		connectRabbitMQ.Close()
		channelRabbitMQ.Close()
		return nil, err
	}
	return (<-chan amqp.Delivery)(messages), nil
}

func DelayMessage(RMQServer string, QName string, i interface{}) error {
	connectRabbitMQ, err := amqp.Dial(RMQServer)
	if err != nil {
		return err
	}
	defer connectRabbitMQ.Close()
	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		return err
	}
	defer channelRabbitMQ.Close()
	bytes, err := json.Marshal(i)
	if err != nil {
		return err
	}
	M := bson.M{}
	err = json.Unmarshal(bytes, &M)
	if err != nil {
		return err
	}
	val, ok := M["HoldTime"]
	if !ok {
		return errors.New("cannot find holdtime field")
	}
	HoldTime := int(val.(float64))
	DelayQ := fmt.Sprintf("DelayedQ_%d", HoldTime)
	HoldTimems := HoldTime * 1000
	_, err = channelRabbitMQ.QueueDeclare(
		DelayQ, // Q Name
		true,   // Durable
		false,  // auto delete
		false,  // exclusive
		false,  // no wait
		amqp.Table{
			"x-dead-letter-exchange":    "",             // Excchange where to send after TTL expire
			"x-dead-letter-routing-key": QName,          // Queue name
			"x-message-ttl":             HoldTimems,     // Q Messages expiration time before moving to other Q
			"x-expires":                 HoldTimems * 2, // Delete Q after x time
		}, // arguments
	)
	if err != nil {
		return err
	}
	message := amqp.Publishing{
		ContentType: "application/json",
		Body:        bytes,
	}
	err = channelRabbitMQ.Publish("", DelayQ, false, false, message)
	if err != nil {
		return err
	}
	return nil
}
