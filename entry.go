package logroll

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	spb "google.golang.org/protobuf/types/known/structpb"
)

var (
	// qualified package name, cached at first use
	stdPackage string

	// Positions in the call stack when tracing to report the calling method
	minimumCallerDepth int

	// Used for caller information initialisation
	callerInitOnce sync.Once
)

const (
	maximumCallerDepth int = 25
	knownSelfFrames    int = 4
)

func init() {
	// start at the bottom of the stack before the package-name cache is primed
	minimumCallerDepth = 1
}

// Defines the key when adding errors using WithError.
var ErrorKey = "error"

// =============================================================== //
// LogEntry
// =============================================================== //
type LogEntry struct {
	Caller string
	Level  logrus.Level
	Time   time.Time
	Value  interface{}
}

// -------------------------------------------------------------- //
// Decode
// ---------------------------------------------------------------//
func (e *LogEntry) Decode(frame []byte) error {
	ival, _ := spb.NewValue(nil)
	err := ival.UnmarshalJSON(frame)
	if err != nil {
		return err
	}
	switch lval := ival.GetKind().(type) {
	case *spb.Value_ListValue:
		for i, value := range lval.ListValue.Values {
			switch i {
			case 0:
				x := int64(value.GetNumberValue())
				e.Time = time.UnixMicro(x)
			case 1:
				x := value.GetStringValue()
				if e.Level, err = logrus.ParseLevel(x); err != nil {
					e.Level = logrus.InfoLevel
				}
			case 2:
				e.Value = value.GetStringValue()
			case 3:
				e.Caller = value.GetStringValue()
			}
		}
	default:
		return fmt.Errorf("invalid structpb kind : %T", ival)
	}
	return nil
}

// -------------------------------------------------------------- //
// Encode
// ---------------------------------------------------------------//
func (e *LogEntry) Encode() ([]byte, error) {
	dset, _ := spb.NewValue([]any{
		e.Time.UnixMicro(),
		e.Level.String(),
		e.Value,
		e.Caller,
	})
	return dset.MarshalJSON()
}

// -------------------------------------------------------------- //
// reset
// ---------------------------------------------------------------//
func (e *LogEntry) reset() {
	e.Caller = ""
	e.Time = time.Time{}
	e.Level = 0
	e.Value = nil
}

// =============================================================== //
// LogWriter
// =============================================================== //
type LogWriter interface {
	Write(*LogEntry) error
	GetWriter() io.Writer
	SetWriter(io.Writer) error
}

// =============================================================== //
// FileWriter
// =============================================================== //
type FileWriter struct {
	allowMultiWrite bool
	writer          io.Writer
	formatter       *TextFormatter
}

// -------------------------------------------------------------- //
// Write
// ---------------------------------------------------------------//
func (fw FileWriter) Write(e *LogEntry) error {
	frame, err := fw.formatter.Format(e)
	if err != nil {
		return err
	}
	_, err = fw.writer.Write(frame)
	return err
}

// -------------------------------------------------------------- //
// GetWriter
// ---------------------------------------------------------------//
func (fw FileWriter) GetWriter() io.Writer {
	return fw.writer
}

// -------------------------------------------------------------- //
// SetWriter
// ---------------------------------------------------------------//
func (fw *FileWriter) SetWriter(w io.Writer) error {
	if fw.allowMultiWrite {
		fw.writer = io.MultiWriter(fw.writer, w)
	} else {
		fw.writer = w
	}
	return nil
}
