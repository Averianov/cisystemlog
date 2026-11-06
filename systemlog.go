package cisystemlog

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"time"
)

var L *Logs

// ### Logger Factory

const (
	ALERT   int32 = 1
	WARNING int32 = 2
	INFO    int32 = 3
	DEBUG   int32 = 4

	ALERT_STRING   string = "ALRT"
	WARNING_STRING string = "WARN"
	INFO_STRING    string = "INFO"
	DEBUG_STRING   string = "DEBG"
	EMPTY_STRING   string = "    "
)

type Logs struct {
	sync.Mutex
	fileLevels map[string]int32 // file name, level log
	filename string
	globalLevel    int32
	fileSize int64
	f        *os.File
}

// Size (Mb) = int64 * 1 000 000 byte. If Size==0 then filelog not created.
// logLevel: {1 - only Alert; 2 - Alert & Warning; 3 - all without Debug; 4 - all}
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

	if L == nil {
		L = &Logs{
			filename: logdir + logname,
			globalLevel:    logLevel,
			fileSize: size * 1000000,
			fileLevels: make(map[string]int32),
		}
	} else {
		L.filename = logdir + logname
		L.globalLevel = logLevel
		L.fileSize = size * 1000000
	}

	L.RemoveLogFile(L.filename+".log", 2)
	L.RemoveLogFile(L.filename+"bkp.zip", 2)
	return L
}

func LevelToString(lvl int32) (string) {
	switch lvl {
	case ALERT:
		return ALERT_STRING
	case WARNING:
		return WARNING_STRING
	case INFO:
		return INFO_STRING
	case DEBUG:
		return DEBUG_STRING
	default:
		return EMPTY_STRING
	}
}

func SetFileLevel(level int32) {
	if L == nil {
		L = CreateLogs("", "", level, 1)
	}

	callerData := getCallerData(0)
	if callerData == nil {
		L.Warning("callerData is nil")
		return
	}
	L.fileLevels[callerData.fileName] = level
	L.Print("Set level:%s to file:%s", addLevelString(level), callerData.fileName)
}

func addLevelString(level int32) string {
	startCode := "\033[96m"
	endCode := "\033[0m"
	switch level {
	case DEBUG:
		startCode = "\033[37m"
	case INFO:
		startCode = "\033[96m"
	case WARNING:
		startCode = "\033[93m"
	case ALERT:
		startCode = "\033[91m"
	}
	return startCode+LevelToString(level)+endCode
}

// Sprint - make log record (date_time source	type event)
func (l *Logs) Sprint(level int32, any ...interface{}) (str string) {
	str = time.Now().Format("2006.01.02_15:04:05")

	if level > DEBUG || level < ALERT {
		str += "			"
	} else {
		cd := getCallerData(1)
		if cd == nil {
			cd = &callerData{
				fileName: "unknown",
				lineNumber: 0,
				funcName: "unknown",
				packageName: "unknown",
			}
		}

		if level < 0 {
			return
		}
		if level > l.globalLevel {
			if reqLvl, ok := l.fileLevels[cd.fileName]; ok {
				if level > reqLvl {
					return
				}
			} else {
				return
			}
		}
		str += " \033[90m\033[47m"                                    // set white background
		str += fmt.Sprintf("%s:%d", cd.fileName, cd.lineNumber)
		str += "\033[0m"
		str += "\t"
	}

	str += addLevelString(level)
	str += " "
	str += fmt.Sprintf(fmt.Sprintf("%v", any[0]), any[1:]...)

	l.WriteLogRecord(str)
	fmt.Println(str)
	return
}

// Print - just print message without any type and logging
func (l *Logs) Print(any ...interface{}) {
	l.Sprint(0, any...)
}

// Debug - message used for debug data
func (l *Logs) Debug(any ...interface{}) {
	l.Sprint(DEBUG, any...)
}

// Info - message used for informing data
func (l *Logs) Info(any ...interface{}) {
	l.Sprint(INFO, any...)
}

// Warning - message used for errors and other warning events
func (l *Logs) Warning(any ...interface{}) {
	l.Sprint(WARNING, any...)
}

// Alert - used for emergency message
func (l *Logs) Alert(any ...interface{}) {
	l.Sprint(ALERT, any...)
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
			l.Warning("%s", err.Error())
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
		l.Warning("Error get file info: %s", err.Error())
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
		l.Warning("retry remove %d", i)
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
