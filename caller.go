package cisystemlog

import (
	"runtime"
	"strings"
	"sync"
)

type callerData struct {
	packageName string
	fileName 	string
	funcName 	string
	lineNumber 	int
}

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
func getCaller(requiredDepth int) *runtime.Frame {
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
		if requiredDepth < 0 {
			return nil
		}

		pkg := getPackageName(f.Function)
		// If the caller isn't part of this package, we're done
		if pkg != logrusPackage {
			if requiredDepth == 0 {
				return &f
			}
			requiredDepth--
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}

// #################################################################
// #################################################################

func getCallerData(depth int) *callerData {
	frame := getCaller(depth)
	if frame == nil {
		return nil
	}
	
	return &callerData{
		fileName: frame.File[strings.LastIndex(frame.File, "/")+1:],
		lineNumber: frame.Line,
		funcName: frame.Function,
		packageName: getPackageName(frame.Function),
	}
}
