package main

import (
	"fmt"
	"net/http"
	"net/url"
)

type Notification struct {
	Log                   string
	NotificationMethod    string
	NotificationRecipient string
	HoldTime              int
	RetryCount            int
	Retried               int
}

type NotificationMethod struct {
	Name     string
	Function func(n Notification) error
}

type NotOperation struct {
	note    Notification
	methods []NotificationMethod
}

func (n *NotOperation) New(Methods []NotificationMethod) {
	n.note = Notification{}
	n.methods = Methods
}

func (n *NotOperation) Execute(note Notification) error {
	n.note = note
	found := false
	var notificationMethod func(n Notification) error
	for _, method := range n.methods {
		if method.Name == n.note.NotificationMethod {
			notificationMethod = method.Function
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("cannot find method: %s", n.note.NotificationMethod)
	}
	err := notificationMethod(n.note)
	if err != nil {
		return err
	}
	return nil
}

func sendTelegramMessage(n Notification) error {
	params := url.Values{
		"chat_id": {n.NotificationRecipient},
		"text":    {n.Log},
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
