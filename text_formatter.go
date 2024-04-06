package logroll

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
)

var baseTimestamp time.Time

func init() {
	baseTimestamp = time.Now()
}

type fieldKey string

type FieldMap map[fieldKey]string

func (f FieldMap) resolve(key fieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}

	return string(key)
}

// =============================================================== //
// TextFormatter formats logs into text
// =============================================================== //
type TextFormatter struct {
	// Force quoting of all values
	ForceQuote bool

	// DisableQuote disables quoting for all values.
	// DisableQuote will have a lower priority than ForceQuote.
	// If both of them are set to true, quote will be forced on all values.
	DisableQuote bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed.
	// The format to use is the same than for time.Format or time.Parse from the standard
	// library.
	// The standard Library already provides a set of predefined format.
	TimestampFormat string

	// PadLevelText Adds padding the level text so that all the levels output at the same length
	// PadLevelText is a superset of the DisableLevelTruncation option
	PadLevelText bool

	// QuoteEmptyFields will wrap empty fields in quotes if true
	QuoteEmptyFields bool

	// Whether the logger's out is to a terminal
	isTerminal bool

	// CallerPrettyfier can be set by the user to modify the content
	// of the function and file keys in the data when ReportCaller is
	// activated. If any of the returned value is the empty string the
	// corresponding key will be removed from fields.
	CallerPrettyfier func(string) (function string, file string)

	// The max length of the level text, generated dynamically on init
	levelTextMaxLength int
}

// -------------------------------------------------------------- //
// init
// ---------------------------------------------------------------//
func (f *TextFormatter) init(writer io.Writer) {
	if writer != nil {
		f.isTerminal = checkIfTerminal(writer)
	}
	// Get the max length of the level text
	for _, level := range logrus.AllLevels {
		levelTextLength := utf8.RuneCount([]byte(level.String()))
		if levelTextLength > f.levelTextMaxLength {
			f.levelTextMaxLength = levelTextLength
		}
	}
}

// -------------------------------------------------------------- //
// Format - renders a single log entry
// ---------------------------------------------------------------//
func (f *TextFormatter) Format(e *LogEntry) ([]byte, error) {
	b := bufferPool.Get()
	b.Reset()

	if !f.DisableTimestamp {
		timestampFormat := f.TimestampFormat
		if timestampFormat == "" {
			timestampFormat = defaultTimestampFormat
		}
		f.appendKeyValue(b, "time", e.Time.Format(timestampFormat))
	}

	f.appendKeyValue(b, "level", e.Level.String())
	f.appendKeyValue(b, "msg", e.Value)

	if e.Caller != "" {
		var funcName, fileName string
		if f.CallerPrettyfier != nil {
			funcName, fileName = f.CallerPrettyfier(e.Caller)
		} else {
			funcName, fileName = getCallerProps(e.Caller)
		}

		if funcName != "" {
			f.appendKeyValue(b, "func", funcName)
		}
		if fileName != "" {
			f.appendKeyValue(b, "file", fileName)
		}
	}

	b.WriteByte('\n')
	frame := b.Bytes()
	b.Reset()
	bufferPool.Put(b)
	return frame, nil
}

// -------------------------------------------------------------- //
// getCallerProps
// ---------------------------------------------------------------//
func getCallerProps(text string) (string, string) {
	caller := strings.SplitN(text, ":", 2)
	if len(caller) == 1 {
		return caller[0], ""
	}
	return caller[0], caller[1]
}

// -------------------------------------------------------------- //
// needsQuoting
// ---------------------------------------------------------------//
func (f *TextFormatter) needsQuoting(text string) bool {
	if f.ForceQuote {
		return true
	}
	if f.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	if f.DisableQuote {
		return false
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

// -------------------------------------------------------------- //
// appendKeyValue
// ---------------------------------------------------------------//
func (f *TextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
}

// -------------------------------------------------------------- //
// AppendValue
// ---------------------------------------------------------------//
func (f *TextFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}
