package logroll

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
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

	// The logging level the logger should log at. This is typically (and defaults
	// to) `logroll.Info`, which allows Info(), Warn(), Error() and Fatal() to be
	// logged.
	level Level
	// Used to sync writing to the log. Locking is enabled by Default
	// mu MutexWrap
	// Reusable empty entry
	entryPool sync.Pool
	// Function to exit the application, defaults to `os.Exit()`
	exitFunc exitFunc

	reportCaller bool

	traceLog TraceLog

	tokenCh AtomicWrite

	writer  LogWriter
	rWriter LogWriter
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
func (f *LogFile) addEntry(level Level, format string, args ...interface{}) {
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

	err := f.writer.Write(e)
	if err != nil {
		f.traceLog.Debugf("LogFile LogWriter error : %v", err) //nolint
		f.Printf("LogFile LogWriter error : %v", err)
	}

	if f.rWriter != nil {
		err = f.rWriter.Write(e)
		if err != nil {
			f.traceLog.Debugf("LogFile LogWriter error : %v", err) //nolint
			f.Printf("LogFile LogWriter error : %v", err)
		}
	}

	f.putEntry(e)

	if err != nil {
		f.traceLog.Debugf("LogFile LogWriter error : %v", err) //nolint
		f.Printf("LogFile entry encoding error : %v", err)
		return
	}

	// replace the token
	f.tokenCh.set(token)
}

// -------------------------------------------------------------- //
// SetTrace
// ---------------------------------------------------------------//
func (f *LogFile) SetTrace(running bool) {
	f.traceLog.running = running
}

// -------------------------------------------------------------- //
// DumpTrace
// ---------------------------------------------------------------//
func (f *LogFile) DumpTrace() error {
	return f.traceLog.Dump(f.writer.GetWriter())
}

// -------------------------------------------------------------- //
// Flush
// ---------------------------------------------------------------//
func (f *LogFile) Flush() {
	if f.rWriter == nil {
		return
	}
	token := f.tokenCh.get()
	e := f.getEntry(InfoLevel)
	e.Value = "__flush__"
	err := f.rWriter.Write(e)
	if err != nil {
		f.traceLog.Debugf("LogFile LogWriter error : %v", err) //nolint
		f.Printf("LogFile LogWriter error : %v", err)
	}
	f.tokenCh.set(token)
}

// -------------------------------------------------------------- //
// Writer
// ---------------------------------------------------------------//
func (f *LogFile) Writer() io.Writer {
	return f.writer.GetWriter()
}

// -------------------------------------------------------------- //
// getEntry
// ---------------------------------------------------------------//
func (f *LogFile) getEntry(level Level) *LogEntry {
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
func (f *LogFile) SetLevel(level Level) {
	f.level = level
}

// -------------------------------------------------------------- //
// SetRemoteWriter
// ---------------------------------------------------------------//
func (f *LogFile) SetRemoteWriter(writer LogWriter) {
	token := f.tokenCh.get()
	f.rWriter = writer
	f.tokenCh.set(token)
}

// -------------------------------------------------------------- //
// SetWriter
// ---------------------------------------------------------------//
func (f *LogFile) SetWriter(iw interface{}) {
	switch writer := iw.(type) {
	case LogWriter:
		f.writer = writer
	case io.Writer:
		f.writer.SetWriter(writer) //nolint
	}
}

// -------------------------------------------------------------- //
// Closef
// ---------------------------------------------------------------//
func (f *LogFile) Closef(format string, args ...interface{}) {
	f.addEntry(DebugLevel, format, args...)
	f.addEntry(DebugLevel, "__EOF__")
}

// -------------------------------------------------------------- //
// Debugf
// ---------------------------------------------------------------//
func (f *LogFile) Debugf(format string, args ...interface{}) {
	f.addEntry(DebugLevel, format, args...)
}

// -------------------------------------------------------------- //
// Infof
// ---------------------------------------------------------------//
func (f *LogFile) Infof(format string, args ...interface{}) {
	f.addEntry(InfoLevel, format, args...)
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
	f.addEntry(WarnLevel, format, args...)
}

// -------------------------------------------------------------- //
// Error
// ---------------------------------------------------------------//
func (f *LogFile) Error(err error) {
	f.addEntry(ErrorLevel, err.Error())
}

// -------------------------------------------------------------- //
// Errorf
// ---------------------------------------------------------------//
func (f *LogFile) Errorf(format string, args ...interface{}) {
	f.addEntry(ErrorLevel, format, args...)
}

// -------------------------------------------------------------- //
// Fatal
// ---------------------------------------------------------------//
func (f *LogFile) Fatal(err error) {
	f.addEntry(FatalLevel, err.Error())
}

// -------------------------------------------------------------- //
// Fatalf
// ---------------------------------------------------------------//
func (f *LogFile) Fatalf(format string, args ...interface{}) {
	f.addEntry(FatalLevel, format, args...)
	os.Exit(1)
}

// -------------------------------------------------------------- //
// Panicf
// ---------------------------------------------------------------//
func (f *LogFile) Panicf(format string, args ...interface{}) {
	f.addEntry(PanicLevel, format, args...)
	// panic(fmt.Sprintf(format, args...))
}

// -------------------------------------------------------------- //
// Tracef
// ---------------------------------------------------------------//
func (f *LogFile) Tracef(format string, args ...interface{}) {
	f.addEntry(TraceLevel, format, args...)
	// panic(fmt.Sprintf(format, args...))
}
