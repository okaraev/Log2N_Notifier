package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type qconfig struct {
	QConnectionString string
	QName             string
}

type webconfig struct {
	QueueConfig []qconfig
}

var GlobalConfig webconfig

var TelegramApiUri string

func throw(err error) {
	if err != nil {
		panic(err)
	}
}

func getEnvVars() error {
	gramPath := os.Getenv("TelegramapiPath")
	if gramPath == "" {
		return fmt.Errorf("cannot get telegramapipath environment variable")
	}
	teleBytes, err := os.ReadFile(gramPath)
	if err != nil {
		return err
	}
	TelegramApiUri = strings.Split(string(teleBytes), "\n")[0]
	PQConStr := os.Getenv("pqconnectionstringPath")
	if PQConStr == "" {
		return fmt.Errorf("cannot get pqconnectionstringpath environment variable")
	}
	PQName := os.Getenv("pqname")
	if PQName == "" {
		return fmt.Errorf("cannot get pqname environment variable")
	}
	SQConStr := os.Getenv("SQConnectionstringPath")
	if SQConStr == "" {
		return fmt.Errorf("cannot get sqconnectionstringpath environment variable")
	}
	SQName := os.Getenv("sqname")
	if SQName == "" {
		return fmt.Errorf("cannot get sqname environment variable")
	}
	q1 := qconfig{QConnectionString: PQConStr, QName: PQName}
	q2 := qconfig{QConnectionString: SQConStr, QName: SQName}
	qconf := []qconfig{q1, q2}
	wconf := webconfig{qconf}
	GlobalConfig = wconf
	return nil
}

func getMQSecret() error {
	for index, cstring := range GlobalConfig.QueueConfig {
		mqpasscontent, err := os.ReadFile(cstring.QConnectionString)
		if err != nil {
			return err
		}
		mqConnString := strings.Split(string(mqpasscontent), "\n")[0]
		GlobalConfig.QueueConfig[index].QConnectionString = mqConnString
	}
	return nil
}

func main() {
	err := getEnvVars()
	throw(err)
	err = getMQSecret()
	throw(err)
	notificationMethods := []NotificationMethod{}
	method := NotificationMethod{Function: sendTelegramMessage, Name: "Telegram"}
	notificationMethods = append(notificationMethods, method)
	Note := NotOperation{}
	Note.New(notificationMethods)
	myRetrierP := Retrier{}
	myRetrierP.New(ReceiveMessage)
	myRetrierS := Retrier{}
	myRetrierS.New(ReceiveMessage)
	messages, err := myRetrierP.Do(GlobalConfig.QueueConfig[0].QConnectionString, GlobalConfig.QueueConfig[0].QName)
	throw(err)
	messagesfromSecondary, err := myRetrierS.Do(GlobalConfig.QueueConfig[1].QConnectionString, GlobalConfig.QueueConfig[1].QName)
	throw(err)
	forever := make(chan bool)
	go func() {
		for message := range messages {
			notification := Notification{}
			err = json.Unmarshal(message.Body, &notification)
			throw(err)
			err := Note.Execute(notification)
			if err != nil {
				message := fmt.Sprint(err)
				if strings.Contains(message, "cannot find method") {
					log.Println(message)
				} else {
					if notification.Retried < notification.RetryCount {
						notification.Retried++
						err = DelayMessage(GlobalConfig.QueueConfig[0].QConnectionString, GlobalConfig.QueueConfig[0].QName, notification)
						if err != nil {
							log.Println(err)
						}
					} else {
						log.Printf("Skipping after %d retries\nLog: %s\nMethod: %s", notification.Retried, notification.Log, notification.NotificationMethod)
					}
				}
			}
			err = message.Ack(true)
			if err != nil {
				log.Println(err)
				myRetrierP.Open()
			}
		}
	}()
	go func() {
		for message := range messagesfromSecondary {
			notification := Notification{}
			err = json.Unmarshal(message.Body, &notification)
			throw(err)
			err := Note.Execute(notification)
			if err != nil {
				message := fmt.Sprint(err)
				if strings.Contains(message, "cannot find method") {
					log.Println(message)
				} else {
					if notification.Retried < notification.RetryCount {
						notification.Retried++
						err = DelayMessage(GlobalConfig.QueueConfig[1].QConnectionString, GlobalConfig.QueueConfig[1].QName, notification)
						if err != nil {
							log.Println(err)
						}
					} else {
						log.Printf("Skipping after %d retries\nLog: %s\nMethod: %s", notification.Retried, notification.Log, notification.NotificationMethod)
					}
				}
			}
			err = message.Ack(true)
			if err != nil {
				log.Println(err)
				myRetrierS.Open()
			}
		}
	}()
	<-forever
}
