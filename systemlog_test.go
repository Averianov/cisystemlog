package cisystemlog

import (
	"testing"
)

// TestLogging ###############################################
func TestLogging(t *testing.T) {
	CreateLogs("", "", ALERT, 0)
	t.Logf("TestLogging start agent\n")

	var list []int = make([]int, 20000)
	for i, _ := range list {
		L.Alert("####################################################WERYBIGDATA%d######################################################", i)
	}
	//t.Errorf("view test")
}
