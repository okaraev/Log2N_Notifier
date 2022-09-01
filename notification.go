package main

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
)

type Notification struct {
	Log                   string
	NotificationMethod    string
	NotificationRecipient string
	HoldTime              int
	RetryCount            int
	Retried               int
}

type NotificationMethod struct{}

type NotOperation struct {
	Type  reflect.Type
	Value reflect.Value
}

func GetNotificationOperation() NotOperation {
	NO := NotOperation{}
	NO.Type = reflect.TypeOf(&NotificationMethod{})
	NO.Value = reflect.ValueOf(&NotificationMethod{})
	return NO
}

func (n *NotOperation) Execute(note Notification) error {
	found := false
	for i := 0; i < n.Type.NumMethod(); i++ {
		method := n.Type.Method(i)
		if note.NotificationMethod == method.Name {
			found = true
			result := method.Func.Call([]reflect.Value{n.Value, reflect.ValueOf(note)})
			if !result[0].IsNil() {
				return fmt.Errorf("%v", result[0])
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("cannot find method %s", note.NotificationMethod)
	}
	return nil
}

func (n NotificationMethod) Telegram(note Notification) error {
	params := url.Values{
		"chat_id": {note.NotificationRecipient},
		"text":    {note.Log},
	}
	result, err := http.PostForm(TelegramApiUri, params)
	if err != nil {
		return err
	} else if result.StatusCode != 200 {
		message := result.Status
		return fmt.Errorf("couldn't send message: %s", message)
	}
	return nil
}
