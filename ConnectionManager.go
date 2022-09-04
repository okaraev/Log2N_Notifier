package main

import (
	"log"
	"time"
)

type Retrier struct {
	Status      string        // Retrier Status
	HoldTime    time.Duration // Time duration after which Operation will retry. If Operation fails hold time multiplies until 360 seconds
	MaxHoldTime time.Duration
	Operation   func() (<-chan interface{}, error)
}

func GetRetrierOverloadInstance(Func func() (<-chan interface{}, error)) Retrier {
	r := Retrier{}
	r.HoldTime = 10 * time.Second
	r.MaxHoldTime = 360 * time.Second
	r.Status = "Closed"
	r.Operation = Func
	return r
}

func GetRetrierDefaultInstance() Retrier {
	r := Retrier{}
	r.HoldTime = 10 * time.Second
	r.MaxHoldTime = 360 * time.Second
	r.Status = "Closed"
	r.Operation = ReceiveMessage
	return r
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

func (r *Retrier) Do() (<-chan interface{}, error) {
	result, err := r.Operation()
	commonReceiver := make(chan interface{})
	if err != nil {
		r.Open()
		log.Println(err)
	}
	if r.isClosed() {
		// if status is success starting routine to receive messages
		go func() {
			for message := range result {
				commonReceiver <- message
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
				retryResult, err := r.Operation()
				if err != nil {
					log.Println(err)
					r.Multiply()
				} else {
					// if retry is success setting status to success and starting routine to receive messages
					r.Close()
					go func() {
						for message := range retryResult {
							commonReceiver <- message
						}
						// Ending routine when error happened and setting status to fail
						r.Open()
					}()
				}
			}
		}
	}()
	return commonReceiver, nil
}
