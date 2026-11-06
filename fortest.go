package cisystemlog

// import (
// 	"fmt"
// )

// func init() {
// 	fmt.Println("test init fortest") ///
// 	SetFileLevel(WARNING)
// 	fmt.Println("test init fortest done") ///
// }


func CheckFileLevel() {
	L.Print("input text")
	L.Debug("debug text")
	L.Info("hello test %s", "one")
	L.Info("two")
	L.Warning("hello test %s", "three")
	L.Warning("four")
	L.Alert("hello test %s", "five")
	L.Alert("six")
	L.Alert(uint16(25))
	return
}