package main

import (
	"fmt"
	"testing"
)

func MockNotifierFunction(n Notification) error {
	return nil
}

func TestNotifyMessage(t *testing.T) {
	methods := []NotificationMethod{}
	methods = append(methods, NotificationMethod{Name: "Telegram", Function: MockNotifierFunction})
	note := Notification{Log: "testLog", NotificationMethod: "Telegram", NotificationRecipient: "-591104865"}
	myNote := NotOperation{}
	myNote.New(methods)
	err := myNote.Execute(note)
	if err != nil {
		t.Error("Have to return nil instead of error")
	}
	note2 := Notification{Log: "testLog", NotificationMethod: "MethodWhichDoesNotExist", NotificationRecipient: "-591104865"}
	err = myNote.Execute(note2)
	if err == nil {
		t.Error("Have to return error instead of nil")
	} else if fmt.Sprint(err) != "cannot find method: MethodWhichDoesNotExist" {
		t.Errorf("Have to return 'cannot find method: MethodWhichDoesNotExist', but returns: %s", fmt.Sprint(err))
	}
}
