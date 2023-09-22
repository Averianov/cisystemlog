package systemlog

import (
	"testing"
)

// TestLogging ###############################################
func TestLogging(t *testing.T) {
	l := CreateLogs(true, 50)
	t.Logf("TestLogging start agent\n")

	l.Print("input text")
	l.Info("hello test %s", "one")
	l.Info("two")
	l.Warning("hello test %s", "three")
	l.Warning("four")
	l.Alert("hello test %s", "five")
	l.Alert("six")

	t.Logf("check\n")
	if len(l.alert) > 0 {
		t.Errorf("Error Test TestLogging. Bad length; len(result): %d;\n", len(l.alert))
	} else {
		t.Logf("Ok Test TestLogging\n")
	}

	err := l.RemoveLog(3)
	if err != nil {
		t.Errorf("Error Test TestLogging. Error remove log-file: %s;\n", err.Error())
	}
}
