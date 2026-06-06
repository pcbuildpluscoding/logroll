package test

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	logg "github.com/pcbuildpluscoding/logroll"
)

type tcActor func(*testing.T, FlowRuler, ...interface{}) error
type Testcase struct {
	actor   tcActor
	name    string
	dataKey string
}

var logger = logg.New()

var testcases = flag.String("testcases", "", "comma separated list of testcases to run")

var config = FlowRuler{
	logg.FlowRule{
		"testbookA/all": []string{
			"tca_bufferWriter",
			"tca_logEntry",
			"tca_logFile",
		},
		"tca_logFile": map[string]interface{}{
			"LogPath": "/home/devapps/enterprise/github/logroll/testing/log/tca_logfile.log",
		},
	},
}

// ------------------------------------------------------------------//
// getLogWriter
// ------------------------------------------------------------------//
func getLogWriter(logPath string, truncate bool) (*os.File, error) {
	var (
		err     error
		logFile *os.File
	)
	if truncate {
		logFile, err = os.Create(logPath)
	} else {
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
	if err != nil {
		return nil, fmt.Errorf("openFile create|readWrite error : %v", err)
	}
	return logFile, nil
}

// ----------------------------------------------------------------//
// utils
// ----------------------------------------------------------------//
// -------------------------------------------------------------- //
// getEntry
// ---------------------------------------------------------------//
func getEntry(level logg.Level, format string, args ...interface{}) *logg.LogEntry {
	e := &logg.LogEntry{}
	e.Time = time.Now()
	e.Level = level
	e.Value = fmt.Sprintf(format, args...)
	e.Caller = logg.GetCallerText(true)
	return e
}

// ----------------------------------------------------------------//
// getLogMessage
// ----------------------------------------------------------------//
func getLogMessage(text string) string {
	j := 0
	state := 0
	for i, chr := range text {
		switch state {
		case 0:
			if chr == '"' {
				if text[i-4:i] == "msg=" {
					j = i + 1
					state = 1
				}
			}
		case 1:
			if chr == '"' {
				return text[j:i]
			}
		}
	}
	return ""
}

// ------------------------------------------------------------------//
// getTestbookKeys
// ------------------------------------------------------------------//
func getTestbookKeys(rw FlowRuler, bookId string) ([]string, error) {
	if *testcases == "__all__" {
		if !rw.HasKeys(bookId + "/all") {
			return nil, fmt.Errorf(bookId + "/all does not exist in testConfig dataset")
		}
		return rw.StringList(bookId + "/all"), nil
	}
	return strings.Split(*testcases, ","), nil
}

// ----------------------------------------------------------------//
// stringify
// ----------------------------------------------------------------//
func stringify(x interface{}, darg ...string) string {
	d := ","
	if darg != nil {
		d = darg[0]
	}
	switch y := x.(type) {
	case []string:
		return strings.Join(y, d)
	case []interface{}:
		return strings.Join(toStringList(y), d)
	default:
		return ""
	}
}

// ----------------------------------------------------------------//
// toStringList
// ----------------------------------------------------------------//
func toStringList(x []interface{}) []string {
	result := make([]string, len(x))
	for i, ival := range x {
		result[i], _ = ival.(string)
	}
	return result
}

// ------------------------------------------------------------------//
// setLogger
// ------------------------------------------------------------------//
func setLogger() error {

	logPath := "./log/testing.log"

	var err error
	logger, err = logg.WithFile(logPath, logg.DebugLevel)

	return err
}

// ==================================================================//
// FlowRuler
// ==================================================================//
type FlowRuler struct {
	logg.FlowRule
}

// ------------------------------------------------------------------//
// SubNode
// ------------------------------------------------------------------//
func (r FlowRuler) SubNode(key string, err *error) *FlowRuler {
	if !r.HasKeys(key) {
		*err = fmt.Errorf("%s does not exist", key)
		return &FlowRuler{logg.FlowRule{}}
	}
	switch sn := r.FlowRule[key].(type) {
	case map[string]interface{}:
		return &FlowRuler{
			logg.FlowRule(sn),
		}
	default:
		*err = fmt.Errorf("%s is not a map[string]interface type", key)
		return &FlowRuler{logg.FlowRule{}}
	}
}
