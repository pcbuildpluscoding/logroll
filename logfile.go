package logroll

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type exitFunc func(int)

// ==================================================================//
// AtomicWrite
// ==================================================================//
type AtomicWrite struct {
	stateCh chan Void
}

// -------------------------------------------------------------- //
// get
// ---------------------------------------------------------------//
func (a *AtomicWrite) get() Void {
	return <-a.stateCh
}

// -------------------------------------------------------------- //
// set
// ---------------------------------------------------------------//
func (a *AtomicWrite) set(token Void) {
	a.stateCh <- token
}

// =============================================================== //
// LogFile
// =============================================================== //
type LogFile struct {
	category FlowRule

	formatter Formatter

	// The logging level the logger should log at. This is typically (and defaults
	// to) `logrus.Info`, which allows Info(), Warn(), Error() and Fatal() to be
	// logged.
	level logrus.Level
	// Used to sync writing to the log. Locking is enabled by Default
	// mu MutexWrap
	// Reusable empty entry
	entryPool sync.Pool
	// Function to exit the application, defaults to `os.Exit()`
	exitFunc exitFunc

	remoteWriter bool

	reportCaller bool

	tokenCh AtomicWrite

	Out  io.Writer
	rOut io.Writer
}

// -------------------------------------------------------------- //
// AddLogCat
// ---------------------------------------------------------------//
func (f *LogFile) AddLogCat(args ...interface{}) {
	var (
		key string
		ok  bool
	)
	for i, arg := range args {
		if i%2 == 0 {
			key, ok = arg.(string)
			if !ok {
				f.Printf("LogFile - category name is a not a string")
				continue
			}
		}
		f.category[key] = arg
	}
}

// -------------------------------------------------------------- //
// addEntry
// ---------------------------------------------------------------//
func (f *LogFile) addEntry(level logrus.Level, format string, args ...interface{}) {
	if level > f.level {
		return
	}
	// only one goroutine can get the token, all other will block
	token := f.tokenCh.get()

	e := f.getEntry(level)
	e.Value = fmt.Sprintf(format, args...)

	if f.reportCaller {
		e.Caller = GetCallerText(false)
	}

	frame, err := f.getBytes(e)

	f.putEntry(e)

	if err != nil {
		f.Printf("LogFile entry encoding error : %v", err)
		return
	}

	_, err = f.Out.Write(frame)
	if err != nil {
		f.Printf("LogFile writer error : %v", err)
	}
	// replace the token
	f.tokenCh.set(token)
}

// -------------------------------------------------------------- //
// getBytes
// ---------------------------------------------------------------//
func (f *LogFile) getBytes(e *LogEntry) ([]byte, error) {
	if f.remoteWriter {
		frame, err := e.Encode()
		if err != nil {
			fmt.Printf("LogEntry encode error : %v\n", err)
			f.remoteWriter = false
		}
		_, err = f.rOut.Write(frame)
		if err != nil {
			fmt.Printf("remote writer error : %v\n", err)
			f.remoteWriter = false
		}
	}

	return f.formatter.Format(e)
}

// -------------------------------------------------------------- //
// getEntry
// ---------------------------------------------------------------//
func (f *LogFile) getEntry(level logrus.Level) *LogEntry {
	e, ok := f.entryPool.Get().(*LogEntry)
	if !ok {
		e = &LogEntry{}
	}
	e.Time = time.Now()
	e.Level = level
	return e
}

// -------------------------------------------------------------- //
// putEntry
// ---------------------------------------------------------------//
func (f *LogFile) putEntry(e *LogEntry) {
	e.reset()
	f.entryPool.Put(e)
}

// -------------------------------------------------------------- //
// SetLevel
// ---------------------------------------------------------------//
func (f *LogFile) SetLevel(level logrus.Level) {
	f.level = level
}

// -------------------------------------------------------------- //
// SetRemoteWriter
// ---------------------------------------------------------------//
func (f *LogFile) SetRemoteWriter(writer io.Writer) {
	f.rOut = writer
	f.remoteWriter = true
}

// -------------------------------------------------------------- //
// SetWriter
// ---------------------------------------------------------------//
func (f *LogFile) SetWriter(writer io.Writer, multiWriter bool) {
	if writer == nil {
		return
	}
	if checkIfTerminal(writer) {
		fr, ok := f.formatter.(*TextFormatter)
		if !ok {
			return
		}
		fr.init(writer)
	} else if multiWriter {
		f.Out = io.MultiWriter(f.Out, writer)
		return
	}
	f.Out = writer
}

// -------------------------------------------------------------- //
// SetFormatter
// ---------------------------------------------------------------//
func (f *LogFile) SetFormatter(fr Formatter) {
	f.formatter = fr
}

// -------------------------------------------------------------- //
// init
// ---------------------------------------------------------------//
func (f *LogFile) init() *LogFile {
	f.entryPool = sync.Pool{
		New: func() any {
			return &LogEntry{}
		},
	}

	fr, ok := f.formatter.(*TextFormatter)
	if !ok {
		return f
	}
	fr.init(f.Out)
	return f
}

// -------------------------------------------------------------- //
// Debugf
// ---------------------------------------------------------------//
func (f *LogFile) Debugf(format string, args ...interface{}) {
	f.addEntry(logrus.DebugLevel, format, args...)
}

// -------------------------------------------------------------- //
// Infof
// ---------------------------------------------------------------//
func (f *LogFile) Infof(format string, args ...interface{}) {
	f.addEntry(logrus.InfoLevel, format, args...)
}

// -------------------------------------------------------------- //
// Printf
// ---------------------------------------------------------------//
func (f *LogFile) Printf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// -------------------------------------------------------------- //
// Warnf
// ---------------------------------------------------------------//
func (f *LogFile) Warnf(format string, args ...interface{}) {
	f.addEntry(logrus.WarnLevel, format, args...)
}

// -------------------------------------------------------------- //
// Error
// ---------------------------------------------------------------//
func (f *LogFile) Error(err error) {
	f.addEntry(logrus.ErrorLevel, err.Error())
}

// -------------------------------------------------------------- //
// Errorf
// ---------------------------------------------------------------//
func (f *LogFile) Errorf(format string, args ...interface{}) {
	f.addEntry(logrus.ErrorLevel, format, args...)
}

// -------------------------------------------------------------- //
// Fatalf
// ---------------------------------------------------------------//
func (f *LogFile) Fatalf(format string, args ...interface{}) {
	f.addEntry(logrus.FatalLevel, format, args...)
	os.Exit(1)
}

// -------------------------------------------------------------- //
// Panicf
// ---------------------------------------------------------------//
func (f *LogFile) Panicf(format string, args ...interface{}) {
	f.addEntry(logrus.PanicLevel, format, args...)
	// panic(fmt.Sprintf(format, args...))
}

// -------------------------------------------------------------- //
// Tracef
// ---------------------------------------------------------------//
func (f *LogFile) Tracef(format string, args ...interface{}) {
	f.addEntry(logrus.TraceLevel, format, args...)
	// panic(fmt.Sprintf(format, args...))
}
