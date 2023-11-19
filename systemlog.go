package systemlog

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	utils "github.com/Averianov/ciutils"
)

var L *Logs

// ### Logger Factory

// ### From Logrus Code #############################################
// #################################################################
var (
	// qualified package name, cached at first use
	logrusPackage string

	// Positions in the call stack when tracing to report the calling method
	minimumCallerDepth int

	// Used for caller information initialisation
	callerInitOnce sync.Once
)

const (
	maximumCallerDepth int = 25
	knownLogrusFrames  int = 4
)

// #################################################################
// #################################################################

type Logs struct {
	sync.Mutex
	caller   *runtime.Frame
	logLevel int32
	logSize  int64
}

const (
	ALERT   string = "ALRT"
	WARNING string = "WARN"
	INFO    string = "INFO"
	DEBUG   string = "DEBG"
)

// ### From Logrus Code #############################################
// #################################################################
// getCaller retrieves the name of the first non-logrus calling function
func getCaller() *runtime.Frame {
	// cache this package's fully-qualified name
	callerInitOnce.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(funcName, "getCaller") {
				logrusPackage = getPackageName(funcName)
				break
			}
		}

		minimumCallerDepth = knownLogrusFrames
	})

	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)

		// If the caller isn't part of this package, we're done
		if pkg != logrusPackage {
			return &f //nolint:scopelint
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}

// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}

// #################################################################
// #################################################################

// Status = true if need more details logs; Size (Mb) = int64 * 1 000 000 byte.\n
// LogLevel: {1 - only Alert; 2 - Alert & Warning; 3 - all without Debug; 4 - all}
func CreateLogs(logLevel int32, size int64) (l *Logs) {
	L = &Logs{
		logLevel: logLevel,
		logSize:  size * 1000000,
	}
	return L
}

func (l *Logs) Print(any ...interface{}) {
	if l.logLevel < 4 {
		return
	}
	fmt.Println(l.Sprint("", any...))
}

func (l *Logs) Debug(any ...interface{}) {
	if l.logLevel < 4 {
		return
	}
	fmt.Println(l.Sprint(DEBUG, any...))
}

func (l *Logs) Info(any ...interface{}) {
	if l.logLevel < 3 {
		return
	}
	fmt.Println(l.Sprint(INFO, any...))
}

func (l *Logs) Warning(any ...interface{}) {
	if l.logLevel < 2 {
		return
	}
	w := l.Sprint(WARNING, any...)
	fmt.Println(w)
	l.WriteLog(w)
}

func (l *Logs) Alert(any ...interface{}) {
	a := l.Sprint(ALERT, any...)
	fmt.Println(a)
	l.WriteLog(a)
}

func (l *Logs) Sprint(mtype string, any ...interface{}) (str string) {
	str = strconv.Itoa(time.Now().Year()) + "." +
		utils.PartDateToStr(int(time.Now().Month())) + "." +
		utils.PartDateToStr(time.Now().Day()) + "_" +
		utils.PartDateToStr(time.Now().Hour()) + ":" +
		utils.PartDateToStr(time.Now().Minute()) + ":" +
		utils.PartDateToStr(time.Now().Second())

	l.caller = getCaller()
	//funcVal := l.caller.Function
	fileVal := fmt.Sprintf("%s:%d", l.caller.File, l.caller.Line)
	if mtype == "" {
		mtype = "\t"
	}
	str += "  " + mtype
	//str += "  " + funcVal[strings.LastIndex(funcVal, "/")+1:]
	str += "  " + fileVal[strings.LastIndex(fileVal, "/")+1:] + ":  "
	str += fmt.Sprintf(fmt.Sprintf("%s", any[0]), any[1:]...)
	//str += "\n"
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
