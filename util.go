package logroll

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// ------------------------------------------------------------------//
// checkIfTerminal
// ------------------------------------------------------------------//
func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return isTerminal(int(v.Fd()))
	default:
		return false
	}
}

// -------------------------------------------------------------- //
// getCaller retrieves the name of the first external calling function
// ---------------------------------------------------------------//
func getCaller() *runtime.Frame {
	// cache this package's fully-qualified name
	callerInitOnce.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(funcName, "getCaller") {
				stdPackage = getPackageName(funcName)
				break
			}
		}

		minimumCallerDepth = knownSelfFrames
	})

	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)

		// If the caller isn't part of this package, we're done
		if pkg != stdPackage {
			return &f //nolint:scopelint
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}

// -------------------------------------------------------------- //
// GetCallerText
// ---------------------------------------------------------------//
func GetCallerText(trimmed bool) string {
	caller := getCaller()
	if trimmed {
		return fmt.Sprintf("%s:%s:%d", trimFunc(caller.Function), trimCaller(caller.File), caller.Line)
	}
	return fmt.Sprintf("%s:%s:%d", caller.Function, caller.File, caller.Line)
}

// -------------------------------------------------------------- //
// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
// ---------------------------------------------------------------//
func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
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
// trimPath - reduces the path of the caller file name
// ------------------------------------------------------------------//
func trimCaller(fileName string) string {
	fitems := strings.Split(fileName, "/")
	size := len(fitems)
	if size <= 2 {
		return fileName
	}
	return fmt.Sprintf("%s/%s", fitems[size-2], fitems[size-1])
}

// ------------------------------------------------------------------//
// trimFunc - reduces the path of the caller function name
// ------------------------------------------------------------------//
func trimFunc(fileName string) string {
	fitems := strings.Split(fileName, ".")
	size := len(fitems)
	if size <= 2 {
		return fileName
	}
	return fmt.Sprintf("%s.%s", fitems[size-2], fitems[size-1])
}

// ==================================================================//
// FlowRule
// ==================================================================//
type FlowRule map[string]interface{}

// ------------------------------------------------------------------//
// Add
// ------------------------------------------------------------------//
func (r FlowRule) Add(key string, value interface{}) {
	r[key] = value
}

// ----------------------------------------------------------------//
// Runware
// ----------------------------------------------------------------//
func (r FlowRule) AsMap() map[string]interface{} {
	return r
}

// ------------------------------------------------------------------//
// Bool
// ------------------------------------------------------------------//
func (r FlowRule) Bool(key string) bool {
	x, _ := r[key].(bool)
	return x
}

// ------------------------------------------------------------------//
// Copy
// ------------------------------------------------------------------//
func (r FlowRule) Copy() FlowRule {
	x := FlowRule{}
	for k, v := range r {
		x.Add(k, v)
	}
	return x
}

// ------------------------------------------------------------------//
// Float
// ------------------------------------------------------------------//
func (r FlowRule) Float(key string) float64 {
	switch x := r[key].(type) {
	case nil:
	case float64:
		return x
	case int:
		return float64(x)
	}
	return 0
}

// ------------------------------------------------------------------//
// HasKeys
// ------------------------------------------------------------------//
func (r FlowRule) HasKeys(keys ...string) bool {
	found := false
	for _, key := range keys {
		if _, found = r[key]; !found {
			return false
		}
	}
	return found
}

// ------------------------------------------------------------------//
// Int
// ------------------------------------------------------------------//
func (r FlowRule) Int(key string) int {
	switch x := r[key].(type) {
	case nil:
	case float64:
		return int(x)
	case int:
		return x
	}
	return 0
}

// ------------------------------------------------------------------//
// Pop
// ------------------------------------------------------------------//
func (r FlowRule) Pop(key string) interface{} {
	x := r[key]
	delete(r, key)
	return x
}

// ------------------------------------------------------------------//
// String
// ------------------------------------------------------------------//
func (r FlowRule) String(key string) string {
	x, _ := r[key].(string)
	return x
}

// ------------------------------------------------------------------//
// StringList
// ------------------------------------------------------------------//
func (r FlowRule) StringList(key string) []string {
	w, found := r[key]
	if !found {
		return []string{}
	}
	switch x := w.(type) {
	case []string:
		return x
	case []interface{}:
		return toStringList(x)
	}
	return []string{}
}

// ------------------------------------------------------------------//
// Value
// ------------------------------------------------------------------//
func (r FlowRule) Value(key string) interface{} {
	x, _ := r[key]
	return x
}

// ==================================================================//
// TraceLog
// ==================================================================//
type TraceLog struct {
	this    bytes.Buffer
	running bool
}

func (t *TraceLog) Debugf(format string, args ...interface{}) error {
	if !t.running {
		return nil
	}
	_, err := fmt.Fprintf(&t.this, format+"\n", args...)
	return err
}

func (t *TraceLog) Dump(w io.Writer) error {
	if w == nil {
		return nil
	}
	_, err := fmt.Fprintln(w, t.this.String())
	t.this.Reset()
	return err
}
