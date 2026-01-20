package systemlog

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var L *Logs

// ### Logger Factory

type Logs struct {
	sync.Mutex
	caller   *runtime.Frame
	filename string
	level    int32
	fileSize int64
	f        *os.File
}

const (
	ALERT   string = "ALRT"
	WARNING string = "WARN"
	INFO    string = "INFO"
	DEBUG   string = "DEBG"
)

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

// #################################################################
// #################################################################

// Size (Mb) = int64 * 1 000 000 byte. If Size==0 then filelog not created.
// LogLevel: {1 - only Alert; 2 - Alert & Warning; 3 - all without Debug; 4 - all}
func CreateLogs(logname, logdir string, logLevel int32, size int64) (l *Logs) {
	if logname == "" {
		logname = "errors"
	}

	if logdir == "" {
		logdir = "./"
	} else {
		runes := []rune(logdir)
		if runes[len(runes)-1] != '/' {
			logdir = logdir + "/"
		}
	}

	L = &Logs{
		filename: logdir + logname,
		level:    logLevel,
		fileSize: size * 1000000,
	}

	L.RemoveLogFile(L.filename+".log", 2)
	L.RemoveLogFile(L.filename+"bkp.zip", 2)
	return L
}

// Print - just print message without any type and logging
func (l *Logs) Print(any ...interface{}) {
	if l.level < 4 {
		return
	}
	fmt.Println(l.Sprint("", any...))
}

// Debug - message used for debug data
func (l *Logs) Debug(any ...interface{}) {
	if l.level < 4 {
		return
	}
	w := l.Sprint("\033[37m"+DEBUG+"\033[0m", any...)
	l.WriteLogRecord(w)
	fmt.Println(w)
}

// Info - message used for informing data
func (l *Logs) Info(any ...interface{}) {
	if l.level < 3 {
		return
	}
	w := l.Sprint("\033[96m"+INFO+"\033[0m", any...)
	l.WriteLogRecord(w)
	fmt.Println(w)
}

// Warning - message used for errors and other warning events
func (l *Logs) Warning(any ...interface{}) {
	if l.level < 2 {
		return
	}
	w := l.Sprint("\033[93m"+WARNING+"\033[0m", any...)
	l.WriteLogRecord(w)
	fmt.Println(w)
}

// Alert - used for emergency message
func (l *Logs) Alert(any ...interface{}) {
	a := l.Sprint("\033[91m"+ALERT+"\033[0m", any...)
	l.WriteLogRecord(a)
	fmt.Println(a)
}

// Sprint - make log record (date_time source	type event)
func (l *Logs) Sprint(mtype string, any ...interface{}) (str string) {
	str = time.Now().Format("2006.01.02_15:04:05")

	if mtype == "\033[96m"+INFO+"\033[0m" {
		str += "			"
	} else {
		l.caller = getCaller()
		if l.caller != nil {
			fileVal := fmt.Sprintf("%s:%d", l.caller.File, l.caller.Line) // get source - file name and line number
			str += " \033[90m\033[47m"                                    // set white background
			str += fileVal[strings.LastIndex(fileVal, "/")+1:]            // get last parsed value
			str += "\033[0m"
			str += "\t"
		} else {
			str += "			"
		}
	}
	//funcVal := l.caller.Function // get function name
	//str += "  " + funcVal[strings.LastIndex(funcVal, "/")+1:]

	if mtype != "" {
		mtype += " "
	}
	str += mtype
	str += fmt.Sprintf(fmt.Sprintf("%v", any[0]), any[1:]...)
	return
}

// WriteLogRecord, i - is what count retries for removing log
func (l *Logs) WriteLogRecord(log string) (err error) {
	if l.fileSize == 0 {
		//
		return
	}
	l.Lock()
	var f *os.File
	if f, err = os.OpenFile(l.filename+".log", os.O_RDWR|os.O_APPEND, 0660); err != nil {
		if f, err = os.Create(l.filename + ".log"); err != nil {
			fmt.Printf("%s\n", err.Error())
			return
		}
	}
	defer func() {
		f.Close()
		l.Unlock()
	}()

	if _, err = f.WriteString(log + `
`); err != nil {
		return
	}

	var fi fs.FileInfo
	fi, err = f.Stat()
	if err != nil {
		fmt.Printf("## WriteLog - Error get file info: %s\n", err.Error())
	} else {
		if fi.Size() > l.fileSize { // 50Mb - максимальный размер лог-файла
			f.Close()
			os.Rename(l.filename+".log", l.filename+"_bkp.log")
			go l.CompressLogs(l.filename + "_bkp.log")
			return
		}
	}
	return
}

// RemoveLogFile, i - is what count retries for removing log
func (l *Logs) RemoveLogFile(filename string, i int) (err error) {
	l.Mutex.Lock()
	err = os.RemoveAll(filename)
	l.Mutex.Unlock()
	if err != nil && i > 0 {
		time.Sleep(100 * time.Millisecond)
		i--
		fmt.Printf("## RemoveLog - retry remove %d\n", i)
		return l.RemoveLogFile(filename, i)
	}
	return
}

// CompressLogs removing old archive file (archive.zip), make new archive file and removing old log file (errors.log)
func (l *Logs) CompressLogs(filename string) (err error) {
	l.RemoveLogFile(filename+".zip", 3)

	var a *os.File
	a, err = os.Create(filename + ".zip")
	if err != nil {
		return
	}
	defer a.Close()

	zipWriter := zip.NewWriter(a)
	defer zipWriter.Close()

	var w io.Writer
	w, err = zipWriter.Create(filename)
	if err != nil {
		return
	}

	var f *os.File
	if f, err = os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0660); err != nil {
		return
	}

	if _, err = io.Copy(w, f); err != nil {
		return
	}

	err = os.RemoveAll(filename)
	l.Alert("logfile %s was archived", filename)
	return
}
