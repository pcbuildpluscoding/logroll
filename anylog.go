package logroll

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// =============================================================== //
// AnyLog
// =============================================================== //
type AnyLog struct {
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

	reportCaller bool

	traceLog TraceLog

	tokenCh AtomicWrite

	writer LogWriter
}

// -------------------------------------------------------------- //
// AddLogCat
// ---------------------------------------------------------------//
func (f *AnyLog) AddLogCat(args ...interface{}) {
	var (
		key string
		ok  bool
	)
	for i, arg := range args {
		if i%2 == 0 {
			key, ok = arg.(string)
			if !ok {
				f.Printf("AnyLog - category name is a not a string")
				continue
			}
		}
		f.category[key] = arg
	}
}

// -------------------------------------------------------------- //
// addEntry
// ---------------------------------------------------------------//
func (f *AnyLog) addEntry(level Level, format string, args ...interface{}) {
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

	if f.writer == nil {
		panic("AnyLog writer is nil")
	}
	err := f.writer.Write(e)
	if err != nil {
		f.traceLog.Debugf("AnyLog LogWriter error : %v", err) //nolint
		f.Printf("AnyLog LogWriter error : %v", err)
	}

	f.putEntry(e)

	if err != nil {
		f.traceLog.Debugf("AnyLog LogWriter error : %v", err) //nolint
		f.Printf("AnyLog entry encoding error : %v", err)
		return
	}

	// replace the token
	f.tokenCh.set(token)
}

// -------------------------------------------------------------- //
// SetTrace
// ---------------------------------------------------------------//
func (f *AnyLog) SetTrace(running bool) {
	f.traceLog.running = running
}

// -------------------------------------------------------------- //
// DumpTrace
// ---------------------------------------------------------------//
func (f *AnyLog) DumpTrace() error {
	if f.writer == nil {
		panic("AnyLog writer is nil")
	}
	return f.traceLog.Dump(f.writer.GetWriter())
}

// -------------------------------------------------------------- //
// Flush
// ---------------------------------------------------------------//
func (f *AnyLog) Flush() {
	if f.writer == nil {
		panic("AnyLog writer is nil")
	}
	token := f.tokenCh.get()
	e := f.getEntry(InfoLevel)
	e.Value = "__flush__"
	err := f.writer.Write(e)
	if err != nil {
		f.traceLog.Debugf("AnyLog LogWriter error : %v", err) //nolint
		f.Printf("AnyLog LogWriter error : %v", err)
	}
	f.tokenCh.set(token)
}

// -------------------------------------------------------------- //
// Writer
// ---------------------------------------------------------------//
func (f *AnyLog) Writer() io.Writer {
	if f.writer == nil {
		return nil
	}
	return f.writer.GetWriter()
}

// -------------------------------------------------------------- //
// getEntry
// ---------------------------------------------------------------//
func (f *AnyLog) getEntry(level Level) *LogEntry {
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
func (f *AnyLog) putEntry(e *LogEntry) {
	e.reset()
	f.entryPool.Put(e)
}

// -------------------------------------------------------------- //
// SetLevel
// ---------------------------------------------------------------//
func (f *AnyLog) SetLevel(level Level) {
	f.level = level
}

// -------------------------------------------------------------- //
// SetRemoteWriter
// ---------------------------------------------------------------//
func (f *AnyLog) SetWriter(writer LogWriter) {
	token := f.tokenCh.get()
	f.writer = writer
	f.tokenCh.set(token)
}

// -------------------------------------------------------------- //
// Closef
// ---------------------------------------------------------------//
func (f *AnyLog) Close() error {
	if f.writer != nil {
		return f.writer.Close()
	}
	return nil
}

// -------------------------------------------------------------- //
// Debugf
// ---------------------------------------------------------------//
func (f *AnyLog) Debugf(format string, args ...interface{}) {
	f.addEntry(DebugLevel, format, args...)
}

// -------------------------------------------------------------- //
// Infof
// ---------------------------------------------------------------//
func (f *AnyLog) Infof(format string, args ...interface{}) {
	f.addEntry(InfoLevel, format, args...)
}

// -------------------------------------------------------------- //
// Printf
// ---------------------------------------------------------------//
func (f *AnyLog) Printf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// -------------------------------------------------------------- //
// Warnf
// ---------------------------------------------------------------//
func (f *AnyLog) Warnf(format string, args ...interface{}) {
	f.addEntry(WarnLevel, format, args...)
}

// -------------------------------------------------------------- //
// Error
// ---------------------------------------------------------------//
func (f *AnyLog) Error(err error) {
	f.addEntry(ErrorLevel, err.Error())
}

// -------------------------------------------------------------- //
// Errorf
// ---------------------------------------------------------------//
func (f *AnyLog) Errorf(format string, args ...interface{}) {
	f.addEntry(ErrorLevel, format, args...)
}

// -------------------------------------------------------------- //
// Fatal
// ---------------------------------------------------------------//
func (f *AnyLog) Fatal(err error) {
	f.addEntry(FatalLevel, err.Error())
}

// -------------------------------------------------------------- //
// Fatalf
// ---------------------------------------------------------------//
func (f *AnyLog) Fatalf(format string, args ...interface{}) {
	f.addEntry(FatalLevel, format, args...)
	os.Exit(1)
}

// -------------------------------------------------------------- //
// Panicf
// ---------------------------------------------------------------//
func (f *AnyLog) Panicf(format string, args ...interface{}) {
	f.addEntry(PanicLevel, format, args...)
	// panic(fmt.Sprintf(format, args...))
}

// -------------------------------------------------------------- //
// Tracef
// ---------------------------------------------------------------//
func (f *AnyLog) Tracef(format string, args ...interface{}) {
	f.addEntry(TraceLevel, format, args...)
	// panic(fmt.Sprintf(format, args...))
}
