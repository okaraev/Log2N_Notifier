package main

import (
	"fmt"
	"testing"
)

func (n NotificationMethod) ExistFunction(note Notification) error {
	return nil
}

func TestNotificationMethod(t *testing.T) {
	method := GetNotificationOperation()
	if method.Type.NumMethod() != 2 {
		t.Error("Have to return 2 instead of", method.Type.NumMethod())
	}
}

func TestNotifyMessage(t *testing.T) {
	method := GetNotificationOperation()
	note := Notification{Log: "testLog", NotificationMethod: "ExistFunction", NotificationRecipient: "-591104865"}
	err := method.Execute(note)
	if err != nil {
		t.Error("Have to return nil instead of error:", err)
	}
	note2 := Notification{Log: "testLog", NotificationMethod: "MethodWhichDoesNotExist", NotificationRecipient: "-591104865"}
	err = method.Execute(note2)
	if err == nil {
		t.Error("Have to return error instead of nil")
	} else if fmt.Sprint(err) != "cannot find method MethodWhichDoesNotExist" {
		t.Errorf("Have to return 'cannot find method MethodWhichDoesNotExist', but returns: %s", fmt.Sprint(err))
	}
}
