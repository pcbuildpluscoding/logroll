package test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	logg "github.com/pcbuildpluscoding/logroll"
	"gotest.tools/v3/assert"
)

// ----------------------------------------------------------------//
// TestLogFile
// ----------------------------------------------------------------//
func TestLogFile(t *testing.T) {

	err := setLogger()
	if err != nil {
		t.Fatalf("set logger error : %v", err)
	}

	tcSlice, err := getTestbookA(config)
	if err != nil {
		t.Fatalf("getTestbookKeys error : %v", err)
	}
	t.Run("Serviceq", func(t *testing.T) {
		for _, tc := range tcSlice {
			if tc.actor == nil {
				t.Fatalf("testcase |%s| is undefined", tc.name)
			}
			err := tc.actor(t, *config.SubNode(tc.dataKey, &err), nil)
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}

// ----------------------------------------------------------------//
// tca_bufferWriter
// ----------------------------------------------------------------//
func tca_bufferWriter(t *testing.T, rw FlowRuler, args ...interface{}) error {
	fmt.Printf("\nrunning tca_logfile ...\n")

	logger := logg.New()
	assert.Assert(t, logger != nil, "assert-0")

	var buf bytes.Buffer
	logger.SetWriter(&buf)

	logger.Warnf("warning message!")
	assert.Equal(t, "warning message!", getLogMessage(buf.String()), "assert-1")

	buf.Reset()
	logger.Debugf("debug message!")
	assert.Equal(t, 0, len(buf.Bytes()), "assert-2")
	fmt.Printf("%s", buf.Bytes())

	logger.SetLevel(logg.DebugLevel)

	buf.Reset()
	logger.Infof("info message!")
	assert.Equal(t, "info message!", getLogMessage(buf.String()), "assert-3")

	buf.Reset()
	logger.Warnf("warning message!")
	assert.Equal(t, "warning message!", getLogMessage(buf.String()), "assert-4")

	fmt.Println("tca_logfile1 is complete")
	return nil
}

// ----------------------------------------------------------------//
// tca_logEntry
// ----------------------------------------------------------------//
func tca_logEntry(t *testing.T, rw FlowRuler, args ...interface{}) error {
	fmt.Printf("\nrunning tca_logEntry ...\n")

	e1 := getEntry(logg.DebugLevel, "this is a LogClient LogEntry test")
	fmt.Printf("LogEntry : %v\n", *e1)
	frame, err := e1.Encode()
	assert.Assert(t, err, "assert-0")

	e2 := logg.LogEntry{}
	err = e2.Decode(frame)
	assert.Assert(t, err, "assert-1")
	fmt.Printf("LogEntry : %v\n", e2)

	assert.Assert(t, e1.Time.Unix() == e2.Time.Unix(), "assert-2")
	assert.Equal(t, e1.Level.String(), e2.Level.String(), "assert-3")
	assert.Assert(t, e1.Value == e2.Value, "assert-4")
	assert.Assert(t, e1.Caller == e2.Caller, "assert-5")
	fmt.Printf("tca_logEntry caller : %v\n", e1.Caller)

	fmt.Println("tca_logEntry is complete")
	return nil
}

// ----------------------------------------------------------------//
// tca_logFile
// ----------------------------------------------------------------//
func tca_logFile(t *testing.T, rw FlowRuler, args ...interface{}) error {
	fmt.Printf("\nrunning tca_logfile ...\n")

	logPath := rw.String("LogPath")
	fmt.Printf("got testing logfile path : %s\n", logPath)

	if _, err := os.Stat(logPath); err == nil {
		err := os.Remove(logPath)
		assert.NilError(t, err, "assert-0")
	}

	logger, err := logg.WithFile(logPath)
	assert.NilError(t, err, "assert-1")

	logger.Warnf("warning message!")
	// debug messages should not be output when the log level == info
	logger.Debugf("debug message!")

	logger.SetLevel(logg.DebugLevel)

	logger.Debugf("debug message!")

	logger.Infof("info message!")

	logger.Errorf("error message!")

	logger.Error(errors.New("another error!"))

	typeName := fmt.Sprintf("%T", logger.Writer())
	assert.Assert(t, typeName == "*io.multiWriter", "assert-2")

	dump, err := os.ReadFile(logPath) //read the content of file
	if err != nil {
		return err
	}
	for i, line := range bytes.Split(dump, []byte("\n")) {
		fmt.Println(string(line))
		switch i {
		case 0:
			assert.Equal(t, "warning message!", getLogMessage(string(line)), "assert-3-%d", i)
		case 1:
			assert.Equal(t, "debug message!", getLogMessage(string(line)), "assert-3-%d", i)
		case 2:
			assert.Equal(t, "info message!", getLogMessage(string(line)), "assert-3-%d", i)
		case 3:
			assert.Equal(t, "error message!", getLogMessage(string(line)), "assert-3-%d", i)
		case 4:
			assert.Equal(t, "another error!", getLogMessage(string(line)), "assert-3-%d", i)
		}
	}

	err = logger.Close()
	assert.NilError(t, err, "assert-4")

	fmt.Println("tca_logfile is complete")
	return nil
}

// ------------------------------------------------	------------------//
// getTestbookA
// ------------------------------------------------------------------//
func getTestbookA(rw FlowRuler) ([]Testcase, error) {
	w, err := getTestbookKeys(rw, "testbookA")
	if err != nil {
		return nil, err
	}
	y := make([]Testcase, len(w))
	for i, z := range w {
		switch strings.TrimSpace(z) {
		case "tca_bufferWriter":
			y[i] = Testcase{actor: tca_bufferWriter, name: "tca_bufferWriter", dataKey: "tca_logFile"}
		case "tca_logFile":
			y[i] = Testcase{actor: tca_logFile, name: "tca_logFile", dataKey: "tca_logFile"}
		case "tca_logEntry":
			y[i] = Testcase{actor: tca_logEntry, name: "tca_logEntry", dataKey: "tca_logFile"}
		default:
			return nil, fmt.Errorf("unknown testcase name : |%s|", z)
		}
	}
	return y, nil
}
