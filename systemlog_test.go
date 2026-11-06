package cisystemlog

import (
	"testing"
)

// TestLogging ###############################################
func TestLogging(t *testing.T) {
	CreateLogs("", "", ALERT, 1)
	t.Logf("TestLogging start agent\n")

	// var list []int = make([]int, 20000)
	// for i, _ := range list {
	// 	L.Alert("####################################################WERYBIGDATA%d######################################################", i)
	// }

	CheckFileLevel()
	t.Logf("check\n")
	t.Errorf("view test")
}
