package main

import (
	"fmt"
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
	gramPath := os.Getenv("telegramapipath")
	PQConStr := os.Getenv("pqconnectionstringpath")
	PQName := os.Getenv("pqname")
	PQServerAddress := os.Getenv("pqserveraddress")
	SQServerAddress := os.Getenv("sqserveraddress")
	SQConStr := os.Getenv("sqconnectionstringpath")
	SQName := os.Getenv("sqname")
	if gramPath == "" {
		return fmt.Errorf("cannot get telegramapipath environment variable")
	}
	teleBytes, err := os.ReadFile(gramPath)
	if err != nil {
		return err
	}
	TelegramApiUri = strings.Split(string(teleBytes), "\n")[0]
	if PQConStr == "" {
		return fmt.Errorf("cannot get pqconnectionstringpath environment variable")
	}
	if PQName == "" {
		return fmt.Errorf("cannot get pqname environment variable")
	}
	if PQServerAddress == "" {
		return fmt.Errorf("cannot get pqserveraddress environment variable")
	}
	if SQConStr == "" {
		return fmt.Errorf("cannot get sqconnectionstringpath environment variable")
	}
	if SQServerAddress == "" {
		return fmt.Errorf("cannot get sqserveraddress environment variable")
	}
	if SQName == "" {
		return fmt.Errorf("cannot get sqname environment variable")
	}
	pqpassbytes, err := os.ReadFile(PQConStr)
	if err != nil {
		return err
	}
	sqpassbytes, err := os.ReadFile(SQConStr)
	if err != nil {
		return err
	}
	pqpass := strings.Split(string(pqpassbytes), "\n")[0]
	sqpass := strings.Split(string(sqpassbytes), "\n")[0]
	pqconnectionstring := fmt.Sprintf("amqp://%s@%s", pqpass, PQServerAddress)
	sqconnectionstring := fmt.Sprintf("amqp://%s@%s", sqpass, SQServerAddress)
	q1 := qconfig{QConnectionString: pqconnectionstring, QName: PQName}
	q2 := qconfig{QConnectionString: sqconnectionstring, QName: SQName}
	qconf := []qconfig{q1, q2}
	wconf := webconfig{qconf}
	GlobalConfig = wconf
	return nil
}

func main() {
	err := getEnvVars()
	throw(err)
	FMP := GetFileManagerDefaultInstance(GlobalConfig.QueueConfig[0])
	FMS := GetFileManagerDefaultInstance(GlobalConfig.QueueConfig[1])
	myRetrierP := GetRetrierOverloadInstance(FMP.StartReceive)
	myRetrierS := GetRetrierOverloadInstance(FMS.StartReceive)
	messages, err := myRetrierP.Do()
	throw(err)
	messagesfromSecondary, err := myRetrierS.Do()
	throw(err)
	forever := make(chan bool)
	go func() {
		for message := range messages {
			err := FMP.Process(message, FMP.Delay)
			if err != nil {
				if strings.Contains(fmt.Sprint(err), "cannot acknowledge") {
					myRetrierP.Open()
				} else {
					panic(err)
				}
			}
		}
	}()
	go func() {
		for message := range messagesfromSecondary {
			err := FMS.Process(message, FMS.Delay)
			if err != nil {
				if strings.Contains(fmt.Sprint(err), "cannot acknowledge") {
					myRetrierS.Open()
				} else {
					panic(err)
				}
			}
		}
	}()
	<-forever
}
