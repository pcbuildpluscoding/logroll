package logroll

import (
	"fmt"
	"os"
)

// ------------------------------------------------------------------//
// Void
// ------------------------------------------------------------------//
type Void struct{}

// =============================================================== //
// Level
// =============================================================== //
type Level int

// These are the different logging levels. You can set the logging level to log
// on your instance of logger, obtained with `logroll.New()`.
const (
	NoLevel Level = iota
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel
	// FatalLevel level. Logs and then calls `logger.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
)

func (level Level) String() string {
	switch level {
	case PanicLevel:
		return "PanicLevel"
	case FatalLevel:
		return "FatalLevel"
	case ErrorLevel:
		return "ErrorLevel"
	case WarnLevel:
		return "WarnLevel"
	case InfoLevel:
		return "InfoLevel"
	case DebugLevel:
		return "DebugLevel"
	case TraceLevel:
		return "TraceLevel"
	default:
		return "NoLevel"
	}
}

// ParseLevel takes a string level and returns the Logroll log level constant.
func ParseLevel(level string) (Level, error) {
	switch level {
	case "PanicLevel":
		return PanicLevel, nil
	case "FatalLevel":
		return FatalLevel, nil
	case "ErrorLevel":
		return ErrorLevel, nil
	case "WarnLevel":
		return WarnLevel, nil
	case "InfoLevel":
		return InfoLevel, nil
	case "DebugLevel":
		return DebugLevel, nil
	case "TraceLevel":
		return TraceLevel, nil
	}

	return NoLevel, fmt.Errorf("|%s| is not a valid logroll Level", level)
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
func New(arg ...Level) *LogFile {
	level := InfoLevel
	if arg != nil {
		level = arg[0]
	}
	writer := &FileWriter{
		allowMultiWrite: true,
		writer:          os.Stdout,
		formatter:       NewTextFormatter(),
	}
	return &LogFile{
		writer:       writer,
		tokenCh:      newAtomicWrite(),
		level:        level,
		reportCaller: true,
		exitFunc:     os.Exit,
	}
}

// ------------------------------------------------------------------//
// WithFile
// ------------------------------------------------------------------//
func WithFile(logPath string, level ...Level) (*LogFile, error) {
	lf := New(level...)
	writer, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return lf, err
	}
	return lf, lf.writer.SetWriter(writer)
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
