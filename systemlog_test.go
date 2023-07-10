package systemlog

import (
	"fmt"
	"testing"
)

// TestLogging ###############################################
func TestLogging(t *testing.T) {
	l := CreateLogs(true, 50)
	fmt.Printf("TestLogging start agent\n")
	resultLA := make(chan string)
	go l.LoggerAgent(resultLA)
	defer func() { l.Close <- true }()

	l.Print("input text")

	l.Info("hello test %s", "one")
	l.Info("two")
	l.Warning("hello test %s", "three")
	l.Warning("four")
	l.Alert("hello test %s", "five")
	l.Alert("six")

	fmt.Printf("check\n")
	if len(l.alert) > 0 {
		t.Errorf("Error Test TestLogging. Bad length; len(result): %d;\n", len(l.alert))
	} else {
		fmt.Printf("Ok Test TestLogging\n")
	}

	err := l.RemoveLog(3)
	if err != nil {
		t.Errorf("Error Test TestLogging. Error remove log-file: %s;\n", err.Error())
	}
}
