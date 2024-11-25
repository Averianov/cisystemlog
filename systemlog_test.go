package systemlog

import (
	"testing"
)

// TestLogging ###############################################
func TestLogging(t *testing.T) {
	l := CreateLogs(4, 50)
	t.Logf("TestLogging start agent\n")

	l.Print("input text")
	l.Debug("debug text")
	l.Info("hello test %s", "one")
	l.Info("two")
	l.Warning("hello test %s", "three")
	l.Warning("four")
	l.Alert("hello test %s", "five")
	l.Alert("six")
	l.Alert(uint16(25))
	t.Errorf("s")
	t.Logf("check\n")

	err := l.RemoveLogFile(3)
	if err != nil {
		t.Errorf("Error Test TestLogging. Error remove log-file: %s;\n", err.Error())
	}
}
