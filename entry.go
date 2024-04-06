package logroll

import (
	"bytes"
	"fmt"
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
				e.Time = time.Unix(x, 0)
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
		e.Time.Unix(),
		e.Level.String(),
		e.Value,
		e.Caller,
	})
	return dset.MarshalJSON()
}

// -------------------------------------------------------------- //
// putEntry
// ---------------------------------------------------------------//
func (e *LogEntry) reset() {
	e.Caller = ""
	e.Time = time.Time{}
	e.Level = 0
	e.Value = nil
}

// ================================================================//
// Packet
// ================================================================//
type Packet struct {
	this     [][]byte
	seqnum   int
	maxSize  int
	currSize int
}

// -------------------------------------------------------------- //
// append
// ---------------------------------------------------------------//
func (p *Packet) append(frame []byte) {
	p.currSize += len(frame)
	p.this = append(p.this, frame)
}

// -------------------------------------------------------------- //
// addEntry
// ---------------------------------------------------------------//
func (p *Packet) addEntry(entry LogEntry) error {
	frame, err := entry.Encode()
	if err != nil {
		return err
	}
	p.currSize += len(frame)
	p.this = append(p.this, frame)
	return nil
}

// -------------------------------------------------------------- //
// bytes
// ---------------------------------------------------------------//
func (p *Packet) bytes() []byte {
	return bytes.Join(p.flush(), []byte("|+|"))
}

// -------------------------------------------------------------- //
// flush
// ---------------------------------------------------------------//
func (p *Packet) flush() [][]byte {
	x := p.this
	p.this = [][]byte{}
	p.currSize = 0
	return x
}

// -------------------------------------------------------------- //
// full
// ---------------------------------------------------------------//
func (p *Packet) full() bool {
	if p.maxSize <= 0 {
		return true
	}
	return p.currSize >= p.maxSize
}

// -------------------------------------------------------------- //
// nextDbKey
// ---------------------------------------------------------------//
func (p *Packet) nextDbKey(logKey string) string {
	p.seqnum += 1
	return fmt.Sprintf("%s/%d", logKey, p.seqnum)
}

// -------------------------------------------------------------- //
// reset
// ---------------------------------------------------------------//
func (p *Packet) reset(maxSize ...int) {
	p.seqnum = 0
	p.this = [][]byte{}
	p.currSize = 0
	if maxSize != nil {
		p.maxSize = maxSize[0]
	}
}

// -------------------------------------------------------------- //
// newPacket
// ---------------------------------------------------------------//
func newPacket(maxSize int) Packet {
	return Packet{
		this:    [][]byte{},
		maxSize: maxSize,
	}
}
