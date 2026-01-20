package systemlog

import (
	"testing"
)

// TestLogging ###############################################
func TestLogging(t *testing.T) {
	l := CreateLogs("", "", 4, 1)
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

	var list []int = make([]int, 20000)
	for i, _ := range list {
		l.Alert("####################################################WERYBIGDATA%d######################################################", i)
	}
}
