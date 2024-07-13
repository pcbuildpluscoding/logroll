package logroll

import (
	"fmt"
	"io"
	"os"
	"time"
)

const defaultTimestampFormat = time.RFC3339

type Void struct{}

// =============================================================== //
// Logger
// =============================================================== //
type Logger interface {
	Close() error
	Debugf(string, ...any)
	Infof(string, ...any)
	Printf(string, ...any)
	Warnf(string, ...any)
	Error(error)
	Errorf(string, ...any)
	Fatal(error)
	Fatalf(string, ...any)
	Panicf(string, ...any)
	Tracef(string, ...any)
	SetLevel(Level)
}

// =============================================================== //
// Formatter
// =============================================================== //
type Formatter interface {
	Format(*LogEntry) ([]byte, error)
}

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
		return "panic"
	case FatalLevel:
		return "fatal"
	case ErrorLevel:
		return "error"
	case WarnLevel:
		return "warn"
	case InfoLevel:
		return "info"
	case DebugLevel:
		return "debug"
	case TraceLevel:
		return "trace"
	default:
		return "invalid"
	}
}

// ParseLevel takes a string level and returns the Logroll log level constant.
func ParseLevel(level string) (Level, error) {
	switch level {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
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
// NewMinimalFormatter
// ------------------------------------------------------------------//
func NewMinimalFormatter() *MinimalFormatter {
	f := &MinimalFormatter{
		PadLevelText:    true,
		TimestampFormat: "2006-01-02 15:04:05.000000"}
	f.init()
	return f
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
		formatter:       NewMinimalFormatter(),
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
// NewFileWriter
// ------------------------------------------------------------------//
func NewFileWriter(writer io.Writer, allow bool, f Formatter) *FileWriter {
	if f == nil {
		f = NewMinimalFormatter()
	}
	return &FileWriter{
		allowMultiWrite: allow,
		writer:          writer,
		formatter:       f,
	}
}

// ------------------------------------------------------------------//
// NewAnyLog
// ------------------------------------------------------------------//
func NewAnyLog(writer LogWriter, arg ...Level) *AnyLog {
	level := InfoLevel
	if arg != nil {
		level = arg[0]
	}
	return &AnyLog{
		tokenCh:      newAtomicWrite(),
		level:        level,
		reportCaller: true,
		writer:       writer,
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
