package logroll

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

// ------------------------------------------------------------------//
//
//	init
//
// ------------------------------------------------------------------//
func init() {
	levelDesc := os.Getenv("EMB_LOG_LEVEL")
	if levelDesc == "" {
		levelDesc = "debug"
	}
	switch strings.ToLower(levelDesc) {
	case "debug":
		logLevel = logrus.DebugLevel
	case "info":
		logLevel = logrus.InfoLevel
	default:
		fmt.Printf("!!!!!! invalid log level : %s !!!!!!\n", logLevel)
		logLevel = logrus.DebugLevel
	}
	_logger = Get()
}

var (
	_logger   *logrus.Logger
	logLevel  logrus.Level
	mutex     sync.Mutex
	fileRegex = regexp.MustCompile(`/`)
	funcRegex = regexp.MustCompile(`\.`)
)

// ------------------------------------------------------------------//
//
//	trimPath - reduces the path of the caller file name
//
// ------------------------------------------------------------------//
func trimCaller1(fileName string, lineNum int) string {
	fitems := fileRegex.Split(fileName, -1)
	size := len(fitems)
	if size <= 2 {
		return fileName
	}
	return fmt.Sprintf("%s/%s:%d", fitems[size-2], fitems[size-1], lineNum)
}

// ------------------------------------------------------------------//
//
//	trimFunc - reduces the path of the caller function name
//
// ------------------------------------------------------------------//
func trimFunc1(fileName string) string {
	fitems := funcRegex.Split(fileName, -1)
	size := len(fitems)
	if size <= 2 {
		return fileName
	}
	return fmt.Sprintf("%s.%s", fitems[size-2], fitems[size-1])
}

// ------------------------------------------------------------------//
//
//	GetLogLevel
//
// ------------------------------------------------------------------//
func GetLogLevel() logrus.Level {
	return logLevel
}

// ------------------------------------------------------------------//
//
//	levelAllSet
//
// ------------------------------------------------------------------//
type Void struct{}

func allLevelsSet() map[logrus.Level]Void {
	return map[logrus.Level]Void{
		logrus.InfoLevel:  {},
		logrus.DebugLevel: {},
		logrus.ErrorLevel: {},
		logrus.WarnLevel:  {},
		logrus.FatalLevel: {},
		logrus.PanicLevel: {},
	}
}

// ------------------------------------------------------------------//
//
//	filterLevels
//
// ------------------------------------------------------------------//
func filterLevels(mode string, levelKeys ...string) []logrus.Level {
	switch {
	case mode == "include" && levelKeys != nil:
		levels := make([]logrus.Level, len(levelKeys))
		for i, key := range levelKeys {
			var level logrus.Level
			switch key {
			case "info":
				level = logrus.InfoLevel
			case "debug":
				level = logrus.DebugLevel
			case "error":
				level = logrus.ErrorLevel
			case "warn":
				level = logrus.WarnLevel
			case "fatal":
				level = logrus.FatalLevel
			case "panic":
				level = logrus.PanicLevel
			}
			levels[i] = level
		}
		return levels
	case mode == "exclude" && levelKeys != nil:
		levelSet := allLevelsSet()
		for _, key := range levelKeys {
			switch key {
			case "info":
				delete(levelSet, logrus.InfoLevel)
			case "debug":
				delete(levelSet, logrus.DebugLevel)
			case "error":
				delete(levelSet, logrus.ErrorLevel)
			case "warn":
				delete(levelSet, logrus.WarnLevel)
			case "fatal":
				delete(levelSet, logrus.FatalLevel)
			case "panic":
				delete(levelSet, logrus.PanicLevel)
			}
		}
		levels := make([]logrus.Level, len(levelSet))
		i := 0
		for level := range levelSet {
			levels[i] = level
			i++
		}
		return levels
	default:
		return logrus.AllLevels
	}
}

// ------------------------------------------------------------------//
//
//	Get
//
// ------------------------------------------------------------------//
func Get() *logrus.Logger {
	mutex.Lock()
	defer mutex.Unlock()

	if _logger != nil {
		return _logger
	}
	_logger = getDefLogger()
	return _logger
}

// ------------------------------------------------------------------//
//
//	getDefLogger
//
// ------------------------------------------------------------------//
func getDefLogger(altWriter ...io.Writer) *logrus.Logger {
	prettyfier := func(f *runtime.Frame) (string, string) {
		callerTxt := trimCaller1(f.File, f.Line)
		funcTxt := trimFunc1(f.Function)
		return funcTxt, callerTxt
	}
	formatter := &logrus.TextFormatter{
		CallerPrettyfier:          prettyfier,
		DisableTimestamp:          false,
		DisableLevelTruncation:    true,
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		PadLevelText:              true,
		TimestampFormat:           "2006-01-02 15:04:05.000000"}
	var writer io.Writer
	writer = os.Stdout
	if altWriter != nil && altWriter[0] != nil {
		writer = altWriter[0]
	}
	return &logrus.Logger{
		Out:          writer,
		Formatter:    formatter,
		Hooks:        make(logrus.LevelHooks),
		Level:        logLevel,
		ReportCaller: true,
	}
}

// ------------------------------------------------------------------//
//
//	getLogWriter
//
// ------------------------------------------------------------------//
func getLogWriter(logPath string) (*os.File, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("openFile create|readWrite error : %v", err)
	}
	err = redirectStderr(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to redirect stderr to file: %v", err)
	}
	return logFile, nil
}

// ------------------------------------------------------------------//
//
//	WithFile
//
// ------------------------------------------------------------------//
func WithFile(logPath string, notTeedWithStdout ...bool) (*logrus.Logger, error) {
	mutex.Lock()
	defer mutex.Unlock()

	logger := getDefLogger()

	logfile, err := getLogWriter(logPath)
	if err != nil {
		return logger, err
	}

	if notTeedWithStdout != nil {
		logger.Out = logfile
		return logger, nil
	}

	logger.Out = io.MultiWriter(os.Stdout, logfile)
	return logger, nil
}

// ------------------------------------------------------------------//
//
//	WithWriter
//
// ------------------------------------------------------------------//
func WithWriter(writer io.Writer) *logrus.Logger {
	mutex.Lock()
	defer mutex.Unlock()

	return getDefLogger(writer)
}

// ------------------------------------------------------------------//
//
//	redirectStderr
//
// ------------------------------------------------------------------//
func redirectStderr(f *os.File) error {
	return syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
}

// ------------------------------------------------------------------//
// NewTextFormatter
// ------------------------------------------------------------------//
func NewTextFormatter() *TextFormatter {
	prettyfier := func(text string) (string, string) {
		funcTxt, callerTxt := getCallerProps(text)
		return trimFunc(funcTxt), trimCaller(callerTxt)
	}
	return &TextFormatter{
		CallerPrettyfier: prettyfier,
		DisableTimestamp: false,
		FullTimestamp:    true,
		PadLevelText:     true,
		TimestampFormat:  "2006-01-02 15:04:05.000000"}
}

// ------------------------------------------------------------------//
// New
// ------------------------------------------------------------------//
func New(arg ...logrus.Level) *LogFile {
	level := logrus.DebugLevel
	if arg != nil {
		level = arg[0]
	}
	return &LogFile{
		Out:          os.Stdout,
		tokenCh:      newAtomicWrite(),
		formatter:    NewTextFormatter(),
		level:        level,
		reportCaller: true,
		exitFunc:     os.Exit,
	}
}

// ------------------------------------------------------------------//
// NewFile
// ------------------------------------------------------------------//
func NewFile(level logrus.Level, logPath string) (*LogFile, error) {
	lf := New(level)
	writer, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	lf.Out = writer
	lf.traceLog = TraceLog{running: false}
	return lf, err
}

// --------------------	------------------------------------------ //
// newAtomicWrite
// ---------------------------------------------------------------//
func newAtomicWrite() AtomicWrite {
	a := AtomicWrite{
		stateCh: make(chan Void, 1),
	}
	a.stateCh <- Void{}
	return a
}
