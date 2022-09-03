package main

import (
	"fmt"
	"testing"
)

var methodCount int = 3

func (n NotificationMethod) ExistSuccessFunction(note Notification) error {
	return nil
}

func (n NotificationMethod) ExistFailFunction(note Notification) error {
	return fmt.Errorf("cannot deliver your notification")
}

func TestNotificationMethod(t *testing.T) {
	method := GetNotificationOperation()
	if method.Type.NumMethod() != methodCount {
		t.Errorf("Have to return %d instead of %v", methodCount, method.Type.NumMethod())
	}
}

func TestNotifyMessage(t *testing.T) {
	method := GetNotificationOperation()
	note := Notification{Log: "testLog", NotificationMethod: "ExistSuccessFunction", NotificationRecipient: "-591104865"}
	err := method.Execute(note)
	if err != nil {
		t.Error("Have to return nil instead of error:", err)
	}
	note = Notification{Log: "testLog", NotificationMethod: "ExistFailFunction", NotificationRecipient: "-591104865"}
	err = method.Execute(note)
	if err == nil {
		t.Error("Have to return 'cannot deliver your notification' instead of nil")
	} else {
		if fmt.Sprint(err) != "cannot deliver your notification" {
			t.Errorf("Have to return 'cannot deliver your notification' instead of: %v", err)
		}
	}
	note2 := Notification{Log: "testLog", NotificationMethod: "MethodWhichDoesNotExist", NotificationRecipient: "-591104865"}
	err = method.Execute(note2)
	if err == nil {
		t.Error("Have to return error instead of nil")
	} else if fmt.Sprint(err) != "cannot find method MethodWhichDoesNotExist" {
		t.Errorf("Have to return 'cannot find method MethodWhichDoesNotExist', but returns: %s", fmt.Sprint(err))
	}
}
