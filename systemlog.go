package systemlog

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	utils "github.com/Averianov/ciutils"
)

var L *Logs

// ### Logger Factory

type Logs struct {
	sync.Mutex
	logLevel int32
	Close    chan bool
	logSize  int64
	Status   bool
}

const (
	ALERT   string = "ALERT"
	WARNING string = "WARNING"
	INFO    string = "INFO"
	DEBUG   string = "DEBUG"
)

// Status = true if need more details logs; Size (Mb) = int64 * 1 000 000 byte.\n
// LogLevel: {1 - only Alert; 2 - Alert & Warning; 3 - all without Debug; 4 - all}
func CreateLogs(status bool, logLevel int32, size int64) (l *Logs) {
	L = &Logs{
		logLevel: logLevel,
		logSize:  size * 1000000,
		Status:   status,
	}
	return L
}

func (l *Logs) Print(val interface{}, any ...interface{}) {
	if l.logLevel < 4 {
		return
	}
	fmt.Println(l.Sprint("", val, any...))
}

func (l *Logs) Debug(val interface{}, any ...interface{}) {
	if l.logLevel < 4 {
		return
	}
	fmt.Println(l.Sprint(DEBUG, val, any...))
}

func (l *Logs) Info(val interface{}, any ...interface{}) {
	if l.logLevel < 3 {
		return
	}
	fmt.Println(l.Sprint(INFO, val, any...))
}

func (l *Logs) Warning(val interface{}, any ...interface{}) {
	if l.logLevel < 2 {
		return
	}
	w := l.Sprint(WARNING, val, any...)
	fmt.Println(w)
	l.WriteLog(w)
}

func (l *Logs) Alert(val interface{}, any ...interface{}) {
	a := l.Sprint(ALERT, val, any...)
	fmt.Println(a)
	l.WriteLog(a)
}

func (l *Logs) Sprint(mtype string, fnc interface{}, any ...interface{}) (str string) {
	var t []interface{}
	var s interface{}
	funcName := l.GetFunctionName(fnc)
	//fmt.Printf("## Sprint - funcName: %s\n", funcName)
	if funcName != "" {
		mtype = mtype + ":" + funcName
		if len(any) > 0 {
			s = any[0]
			t = append(t, any[1:]...)
		}
	} else {
		s = fnc
		t = append(t, any...)
	}

	str = strconv.Itoa(time.Now().Year()) + "." +
		utils.PartDateToStr(int(time.Now().Month())) + "." +
		utils.PartDateToStr(time.Now().Day()) + "_" +
		utils.PartDateToStr(time.Now().Hour()) + ":" +
		utils.PartDateToStr(time.Now().Minute()) + ":" +
		utils.PartDateToStr(time.Now().Second()) + "_" + mtype + "= "

	str = str + fmt.Sprintf(s.(string), t...)
	return
}

func (l *Logs) WriteLog(log string) (err error) {
	l.Lock()
	var f *os.File
	if f, err = os.OpenFile("errors.log", os.O_RDWR|os.O_APPEND, 0660); err != nil {
		//fmt.Printf("## WriteLog - Not Opened, err: %s\n", err.Error())
		if f, err = os.Create("errors.log"); err != nil {
			//fmt.Printf("## WriteLog - Cannot Created, err: %s\n", err.Error())
			return
		}
	}
	defer func() {
		f.Close()
		l.Unlock()
	}()

	var fi fs.FileInfo
	fi, err = f.Stat()
	if err != nil {
		fmt.Printf("## WriteLog - Error get file info: %s\n", err.Error())
	} else {
		//l.Print("WriteLog - длина файла: %d bytes", fi.Size())
		if fi.Size() > l.logSize { // 50Mb - максимальный размер лог-файла
			f.Close()
			l.CompressLog()
			return l.WriteLog(log)
		}
	}

	if _, err = f.WriteString(log + `
`); err != nil {
		return
	}
	return
}

// what count retries for removing log
func (l *Logs) RemoveLog(i int) (err error) {
	//fmt.Printf("## RemoveLog - remove 'errors.log'\n")
	err = os.RemoveAll("errors.log")
	if err != nil && i > 0 {
		time.Sleep(100 * time.Millisecond)
		i--
		fmt.Printf("## RemoveLog - retry remove %d\n", i)
		return l.RemoveLog(i)
	}
	return
}

// Удаляет старый архив, создаёт новый архив из лог-файла и удаляет старый лог-файл
func (l *Logs) CompressLog() (err error) {
	os.RemoveAll("archive.zip")

	var a *os.File
	a, err = os.Create("archive.zip")
	if err != nil {
		return
	}
	defer a.Close()

	zipWriter := zip.NewWriter(a)
	defer zipWriter.Close()

	var w io.Writer
	w, err = zipWriter.Create("errors.log")
	if err != nil {
		return
	}

	var f *os.File
	if f, err = os.OpenFile("errors.log", os.O_RDWR|os.O_APPEND, 0660); err != nil {
		return
	}

	if _, err = io.Copy(w, f); err != nil {
		return
	}

	err = os.RemoveAll("errors.log")
	return
}

func (l *Logs) GetFunctionName(i interface{}) (funcName string) {
	switch strings.Contains(reflect.TypeOf(i).String(), "func") {
	//switch i.(type) {
	// case i.(type):
	// 	funcName = runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// case "func()":
	// 	funcName = runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	case true:
		//fmt.Printf("## GetFunctionName - start check - %T\n", i)
		funcName = runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
		//fmt.Printf("## GetFunctionName - %s\n", funcName)
	}
	return
}
