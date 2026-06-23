/*
   Copyright 2026 Torsten Foertsch

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
 */

// Package log implements a simple logging package with a hierarchical
// logger tree. Each logger can have children, and configuration changes
// propagate to descendants. The package is thread-safe; all exported
// methods are safe for concurrent use.
//
// Each logger has an optional topic. The idea is that this is the main
// distinction between loggers. Think of the topic as subsystem or part
// of the code.
//
// Please check out the provided examples to get a better understanding.
//
// A log message consists of a timestamp, a log level, an optional topic,
// an optional code location and the actual message.
//
// The package defines the following log levels:
// NOTICE (printed always), PANIC (print and exit 1), ERROR, WARN, INFO,
// DEBUG, DEBG2, DEBG3, DEBG4, DEBG5
//
// The root logger comes with the folowing properties: log level = WARN
// (a message logged with a higher level will NOT be printed); no topic;
// the output is sent to os.Stderr; the timeformat is
// 2006-01-02 15:04:05.000000; code locations are included in DEBUG and
// above; the location contains the fully qualified file name and
// multiline mode is turned off.
// 
// At the beginning the only existing logger is the root logger. Derived
// loggers can be created using New(). Options passed to New() allow to
// configure the child logger.
//
// Another concept is that of the current logger. It is created using the
// package level New() function or using the SetCurrent() function. The
// current logger is used by all package level functions.
//
// All properties of a logger except the topic can be changed after creation.
// A property change is propagated to all derived loggers recursively. So,
// if you change the log level of the root logger, this change is inherited
// by all other loggers.
package log

import (
	"sync"
	"os"
	"io"
	"fmt"
	"time"
	"strings"
	"slices"
	"runtime"
	"path/filepath"
	"unsafe"
)

// Note a logger where prev==&self, is considered closed.
// prev==nil means it's the root logger. The rest builds a tree where
// prev points to the parent and derived contains all the kids.

// Logger represents a node in a hierarchical logger tree.
type Logger struct {
	mu      sync.Mutex
	prev    *Logger
	out     io.Writer
	topic   string
	level   Level
	minloc  Level		   // minimum level to print the code location
	nlocdir int			   // how many directory path components to log
	timefmt string
	multiln bool
	derived map[*Logger]struct{}
}

// LogOpts holds optional configuration for creating a new Logger via New.
type LogOpts struct {
	out     *io.Writer
	topic   *string
	level   *Level
	minloc  *Level		   // minimum level to print the code location
	nlocdir *int		   // how many directory path components to log
	timefmt *string
	multiln *bool
}

func (l *Logger) string() string {
	var outs string
	switch x := l.out.(type) {
	case interface {Name() string}:
		outs = x.Name()
	default:
		outs = fmt.Sprintf(`%v`, l.out)
	}
	var kids []string
	for k, _ := range l.derived {
		kids = append(kids, fmt.Sprintf(`%p`, k))
	}
	closed := ""
	if l.isClosed() {closed = "(closed)"}
	return fmt.Sprintf(
		"%p%s:{prev:%p, out:%v, topic:`%v`, level:%v, tmfmt:`%v`, kids[%s]}",
		l, closed, l.prev, outs, l.topic, l.level, l.timefmt,
		strings.Join(kids, `, `),
	)
}

// String satisfies the Stringer interface.
func (l *Logger) String() string {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.string()
}

// the global root logger
var def Logger = Logger{
	level: WARN,
	out: os.Stderr,
	timefmt: "2006-01-02 15:04:05.000000",
	minloc: DEBUG,
	nlocdir: -1,
}
// current changes when New() is called as a package function. It is used for
// all package functions. If Close() closes the current logger, current becomes
// the prev of the topmost closed logger.
var curmu sync.Mutex
var current *Logger = &def

// Root returns a pointer to the global root logger.
// The root logger cannot be closed and is always available.
func Root() *Logger {return &def}

// L returns the current package-level logger.
// The current logger is used by all top-level logging functions (e.g.
// Log, Debug). It is protected by a global mutex; use this function to
// obtain a consistent snapshot.
func L() *Logger {curmu.Lock(); defer curmu.Unlock(); return current}

// IsRoot reports whether the logger is the root logger.
func (l *Logger) IsRoot() bool {return l == &def}

// IsCurrent reports whether the logger is the current package-level logger.
func (l *Logger) IsCurrent() bool {return l == L()}

// SetCurrent makes this logger the current package-level logger.
// If l is nil, the root logger becomes current.
func (l *Logger) SetCurrent() {
	curmu.Lock(); defer curmu.Unlock()
	if l == nil {current = &def} else {current = l}
}

// Opt is a function type used to apply optional settings to a LogOpts.
type Opt func(*LogOpts)

// WithTimeFmt returns an Opt that sets the time format string.
// The string is passed to [time.Format] in order to format the time
// stamp.
func WithTimeFmt(f string) Opt {return func(lg *LogOpts) {lg.timefmt = &f}}

// WithTimeFmt returns an Opt that turns on/off multiline mode.
// In multiline mode, line breaks in the log message are detected and
// each line is prefixed with the normal line prefix.
func WithMultiLine(x bool) Opt {return func(lg *LogOpts) {lg.multiln = &x}}

// WithMinLocation returns an Opt that sets the minimum log level for printing
// the caller’s source location. Levels outside [NOTICE+1, DEBG5] disable
// location printing.
func WithMinLocation(lv Level) Opt {return func(lg *LogOpts) {
	// anything outside the normal range turns location off
	if lv > DEBG5 || lv <= NOTICE {lv = DEBG5+1}
	lg.minloc = &lv
}}

// WithLocDirectories returns an Opt that sets the number of directory
// components to include in the printed source location. Negative values
// print the full path.
func WithLocDirectories(n int) Opt {return func(lg *LogOpts) {lg.nlocdir = &n}}

// WithLevel returns an Opt that sets the minimum log level for output.
func WithLevel(lv Level) Opt {return func(lg *LogOpts) {
	lv = ToLevel(int(lv))
	lg.level = &lv
}}

// WithOutput returns an Opt that sets the output writer.
func WithOutput(out io.Writer) Opt {return func(lg *LogOpts) {lg.out = &out}}

// WithTopic returns an Opt that sets the log topic string.
// The topic is prepended in square brackets (e.g. " [mytopic]").
// An empty topic disables the topic prefix.
func WithTopic(tp string) Opt {
	return func(lg *LogOpts) {
		if len(tp) > 0 {tp = fmt.Sprintf(` [%s]`, tp)}
		lg.topic = &tp
	}
}

func (l *Logger) cpy() *Logger {
	l.mu.Lock(); defer l.mu.Unlock()
	new := &Logger{
		prev:    l,
		out:     l.out,
		topic:   l.topic,
		level:   l.level,
		timefmt: l.timefmt,
		minloc:  l.minloc,
		nlocdir: l.nlocdir,
		multiln: l.multiln,
	}
	if l.derived == nil {l.derived = make(map[*Logger]struct{}, 10)}
	l.derived[new] = struct{}{}
	return new
}

// New creates a new child logger by copying the current logger’s configuration
// and applying the given options. The new logger is added to this logger’s
// children set.
func (l *Logger) New(_opts ...Opt) *Logger {
	new := l.cpy()
	opts := &LogOpts{}
	for _, o := range _opts {o(opts)}
	if opts.out != nil {new.out = *opts.out}
	if opts.topic != nil {new.topic = *opts.topic}
	if opts.level != nil {new.level = *opts.level}
	if opts.minloc != nil {new.minloc = *opts.minloc}
	if opts.nlocdir != nil {new.nlocdir = *opts.nlocdir}
	if opts.timefmt != nil {new.timefmt = *opts.timefmt}
	if opts.multiln != nil {new.multiln = *opts.multiln}
	return new
}

// New creates and returns a new logger as a child of the current logger
// and sets it as the new current logger. Shortcut for lg=log.L().New();
// lg.SetCurrent()
func NewC(opts ...Opt) *Logger {
	l := L().New(opts...)
	l.SetCurrent()
	return l
}

// New creates and returns a new logger as a child of the current logger
// without making it the current logger. Shortcut for log.L().New().
func NewK(opts ...Opt) *Logger {return L().New(opts...)}

// New creates and returns a new logger as a child of the root logger
// without making it the current logger. Shortcut for log.Root().New().
func NewR(opts ...Opt) *Logger {return Root().New(opts...)}

func (l *Logger) kids(recursive bool) []*Logger {
	if l.derived == nil {return []*Logger{}}

	// we traverse the kids in order to prevent deadlocks
	list := make([]*Logger, 0, 10)
	for lg, _ := range l.derived {list = append(list, lg)}
	slices.SortFunc(list, func(a, b *Logger) int {
		// The elements of list are the keys in the l.derived map.
		// There is no chance of getting 2 equal elements in the list.
		// So we don't have to cover the == case.
		if uintptr(unsafe.Pointer(a)) < uintptr(unsafe.Pointer(b)) {
			return -1
		} else {
			return 1
		}
	})

	if !recursive {return list}

	for _, lg := range(list) {list = append(list, lg.Kids(true)...)}
	
	return list
}

// Kids returns a list of this logger’s direct children.
// If recursive is true, it also includes all descendants (depth-first).
func (l *Logger) Kids(recursive bool) []*Logger {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.kids(recursive)
}

func (l *Logger) isClosed() bool {return l.prev == l}

// IsClosed reports whether the logger has been closed.
func (l *Logger) IsClosed() bool {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.isClosed()
}

func (l *Logger) _close() *Logger {
	l.mu.Lock(); defer l.mu.Unlock()
	prev := l.prev

	for _, lg := range l.kids(true) {
		lg.mu.Lock()
		if lg.IsCurrent() {prev.SetCurrent()}
		// The root logger cannot be a kid of any other logger.
		// So, we can simply mark this logger as closed. We don't need
		// to perform the lg.IsRoot() check below here.
		lg.prev = lg
		lg.mu.Unlock()
	}
	if l.IsCurrent() {prev.SetCurrent()}
	l.derived = nil
	
	if l.IsRoot() {
		// closing root means reinit
		l.out = os.Stderr
		l.level = WARN
		l.timefmt = "2006-01-02 15:04:05.000000"
		l.derived = nil
		l.topic = ``
		l.minloc = DEBUG
		l.nlocdir = -1
		l.multiln = false
		return l
	} else {
		l.prev = l					// mark as closed except for root
		return prev
	}
}

// Close closes the logger and all its descendants.
// After closing, the logger and its subtree are marked as closed and
// cannot be used. The returned pointer is the previous parent (or nil
// for the root).
// Once a logger has been closed, it will not send any messages anymore.
// The root logger cannot be closed. If closed, it will be reinitialized
// with the default values.
func (l *Logger) Close() *Logger {
	prev := l._close()
	if prev != nil {
		prev.mu.Lock(); defer prev.mu.Unlock()
		if prev.derived != nil {delete(prev.derived, l)}
	}
	return prev
}

// Close closes the current logger and all its descendants.
func Close() {L().Close()}

// SetLevel sets the minimum log level on the current logger and all
// its descendants.
func (l *Logger) SetLevel(lvl Level) {
	lvl = ToLevel(int(lvl))		// range check
	l.Noticef("Setting log level from %[1]v to %[2]v", l.GetLevel(), lvl)
	l.mu.Lock(); defer l.mu.Unlock()

	l.level = lvl
	for _, lg := range l.kids(true) {
		lg.mu.Lock()
		lg.level = lvl
		lg.mu.Unlock()
	}
}

// GetLevel returns the logger’s current minimum log level.
func (l *Logger) GetLevel() Level {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.level
}

// SetLevel sets the minimum log level on the current logger and all
// its descendants.
func SetLevel(lvl Level) {L().SetLevel(lvl)}

// GetLevel returns the current logger’s minimum log level.
func GetLevel() Level {return L().GetLevel()}

// SetMinLocation sets the minimum log level for printing the caller’s
// source location on this logger and all its descendants. Levels outside
// the valid range disable location printing.
func (l *Logger) SetMinLocation(lvl Level) {
	// anything outside the normal range turns location off
	if lvl > DEBG5 || lvl <= NOTICE {
		lvl = DEBG5+1
	}

	l.mu.Lock(); defer l.mu.Unlock()

	l.minloc = lvl
	for _, lg := range l.kids(true) {
		lg.mu.Lock()
		lg.minloc = lvl
		lg.mu.Unlock()
	}
}

// GetMinLocation returns the logger’s current minimum location level.
func (l *Logger) GetMinLocation() Level {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.minloc
}

// SetMinLocation sets the minimum location level on the current logger
// and its descendants.
func SetMinLocation(lvl Level) {L().SetMinLocation(lvl)}

// GetMinLocation returns the current logger’s minimum location level.
func GetMinLocation() Level {return L().GetMinLocation()}

// SetLocDirectories sets the number of directory components shown in
// source locations for this logger and all its descendants.
func (l *Logger) SetLocDirectories(n int) {
	l.mu.Lock(); defer l.mu.Unlock()

	l.nlocdir = n
	for _, lg := range l.kids(true) {
		lg.mu.Lock()
		lg.nlocdir = n
		lg.mu.Unlock()
	}
}

// GetLocDirectories returns the number of directory components shown in
// source locations.
func (l *Logger) GetLocDirectories() int {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.nlocdir
}

// SetLocDirectories sets the directory component count on the current
// logger and its descendants.
func SetLocDirectories(n int) {L().SetLocDirectories(n)}

// GetLocDirectories returns the current logger’s directory component count.
func GetLocDirectories() int {return L().GetLocDirectories()}

// SetMultiLine turns multiline mode on or off for this logger and all its
// descendants.
func (l *Logger) SetMultiLine(ml bool) {
	l.mu.Lock(); defer l.mu.Unlock()

	l.multiln = ml
	for _, lg := range l.kids(true) {
		lg.mu.Lock()
		lg.multiln = ml
		lg.mu.Unlock()
	}
}

// GetMultiLine returns true if multiline mode is on for this logger.
func (l *Logger) GetMultiLine() bool {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.multiln
}

// SetMultiLine turns multiline mode on or off for the current logger and
// all its descendants.
func SetMultiLine(ml bool) {L().SetMultiLine(ml)}

// GetMultiLine returns true if multiline mode is on for the current logger.
func GetMultiLine() bool {return L().GetMultiLine()}

// SetOutput sets the output writer for this logger and all its descendants.
func (l *Logger) SetOutput(out io.Writer) {
	switch x := out.(type) {
	case interface {Name() string}:
		l.Noticef("Setting log output to %s", x.Name())
	case interface {String() string}:
		l.Noticef("Setting log output to %v", out)
	}
	l.mu.Lock(); defer l.mu.Unlock()

	l.out = out
	for _, lg := range l.kids(true) {
		lg.mu.Lock()
		lg.out = out
		lg.mu.Unlock()
	}
}

// GetOutput returns the logger’s output writer.
func (l *Logger) GetOutput() io.Writer {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.out
}

// SetOutput sets the output writer on the current logger and its descendants.
func SetOutput(out io.Writer) {L().SetOutput(out)}

// GetOutput returns the current logger’s output writer.
func GetOutput() io.Writer {return L().GetOutput()}

// SetTimeFmt sets the time format string for this logger and all
// its descendants.
func (l *Logger) SetTimeFmt(f string) {
	l.mu.Lock(); defer l.mu.Unlock()

	l.timefmt = f
	for _, lg := range l.kids(true) {
		lg.mu.Lock()
		lg.timefmt = f
		lg.mu.Unlock()
	}
}

// GetTimeFmt returns the logger’s time format string.
func (l *Logger) GetTimeFmt() string {
	l.mu.Lock(); defer l.mu.Unlock()
	return l.timefmt
}

// SetTimeFmt sets the time format string on the current logger and
// its descendants.
func SetTimeFmt(f string) {L().SetTimeFmt(f)}

// GetTimeFmt returns the current logger’s time format string.
func GetTimeFmt() string {return L().GetTimeFmt()}

var _now = time.Now				// use a variable to allow mocking in testing

// SetNow allows to change this package's notion of now. It's intended to
// be used mainly for testing and debugging purposes. The default
// value is [time.Now].
func SetNow(now_f func() time.Time) func() time.Time {
	old := _now
	_now = now_f
	return old
}

func (l *Logger) prnt(lvl Level, m string, loc string) {
	tm := _now().Format(l.timefmt)
	prfx := lvl.String() + l.topic + loc
	if l.multiln {
		start := 0
		for {
			// Find next '\n' from current position
			idx := strings.IndexByte(m[start:], '\n')
			if idx == -1 {
				// No more newlines – process the remainder (last line)
				line := m[start:]
				// Optionally strip trailing '\r' if present (unlikely)
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				if len(line) > 0 {
					fmt.Fprintln(l.out, tm, prfx, line)
				}
				break
			}
			// line ends at start + idx (the character before '\n')
			end := start + idx
			// Check for '\r' just before the '\n'
			if end > start && m[end-1] == '\r' {
				fmt.Fprintln(l.out, tm, prfx, m[start : end-1])
			} else {
				fmt.Fprintln(l.out, tm, prfx, m[start:end])
			}
			// Move past the '\n'
			start = end + 1
		}
	} else {
		fmt.Fprintln(l.out, tm, prfx, m)
	}
}

var _callers = runtime.Callers	// use this in order to mock
var _nframes = 10				// used for testing
func caller(ncomp int) string {
	// we skip the first item, that's the runtime.Callers() function itself.
	// the [0] entry is now this frame.
	// In the remaining frames we then look for the first frame from a
	// different file.

	pc := make([]uintptr, _nframes)
	nframes := _callers(1, pc)
	if nframes <= 0 {return ``}
	frames := runtime.CallersFrames(pc)
	f, more := frames.Next()
	file := f.File
	for {
		if !more {break}
		f, more = frames.Next()
		if f.File != file {
			if ncomp >= 0 {
				list := strings.Split(
					filepath.Clean(f.File),
					string(filepath.Separator),
				)
				if len(list) > ncomp {
					return fmt.Sprintf(
						` (%s:%d)`,
						strings.Join(
							list[len(list)-ncomp-1:],
							string(filepath.Separator),
						),
						f.Line,
					)
				} 
			}
			return fmt.Sprintf(` (%s:%d)`, f.File, f.Line)
		}

		if !more {break}
	}
	return ``
}

// simplest: print a string

// Log outputs a message at the given level if it meets the logger’s threshold.
// The output includes a timestamp, level, optional topic, optional source
// location, and message. In multiline mode, the message is split into
// lines and each line is prefixed as if it was a log message on its own.
func (l *Logger) Log(lvl Level, m string) {
	l.mu.Lock(); defer l.mu.Unlock()
	if l == l.prev {return}		// logger is closed
	if lvl <= l.level {
		clr := ``
		if lvl >= l.minloc {clr = caller(l.nlocdir)}
		l.prnt(lvl, m, clr)
	}
}

// Log outputs a message at the given level using the current logger.
// See [Log] above.
func Log(lvl Level, m string) {L().Log(lvl, m)}

// formatted output

// Logf outputs a formatted message at the given level.
func (l *Logger) Logf(lvl Level, f string, p ...interface{}) {
	l.mu.Lock(); defer l.mu.Unlock()
	if l == l.prev {return}		// logger is closed
	if lvl <= l.level {
		clr := ``
		if lvl >= l.minloc {clr = caller(l.nlocdir)}
		l.prnt(lvl, fmt.Sprintf(f, p...), clr)
	}
}

// Logf outputs a formatted message at the given level using the current logger.
func Logf(lvl Level, f string, p ...interface{}) {L().Logf(lvl, f, p...)}

// formatted + late execution

// Logl outputs a formatted message similar to [Fogf] with late evaluation
// of arguments. Arguments that are functions of type func() interface{}
// are called to obtain their value.
func (l *Logger) Logl(lvl Level, f string, p ...interface{}) {
	l.mu.Lock(); defer l.mu.Unlock()
	if l == l.prev {return}		// logger is closed
	if lvl <= l.level {
		for i, v := range p {
			if fn, ok := v.(func() interface{}); ok {
				p[i] = fn() // replace element with returned value
			}
		}
		clr := ``
		if lvl >= l.minloc {clr = caller(l.nlocdir)}
		l.prnt(lvl, fmt.Sprintf(f, p...), clr)
	}
}

// Logl outputs a formatted message with late evaluation using the
// current logger.
func Logl(lvl Level, f string, p ...interface{}) {L().Logl(lvl, f, p...)}

// shortcuts: simple string

var _exit = os.Exit

// Notice logs a notice message. Notice messages are always printed.
func (l *Logger) Notice(m string) {l.Log(NOTICE, m)}

// Panic logs a panic message and exits with code 1.
func (l *Logger) Panic(m string) {l.Log(PANIC, m); _exit(1)}

// Error logs an error message.
func (l *Logger) Error(m string) {l.Log(ERROR, m)}

// Warn logs a warning message.
func (l *Logger) Warn(m string)  {l.Log(WARN,  m)}

// Info logs an info message.
func (l *Logger) Info(m string)  {l.Log(INFO,  m)}

// Debug logs a debug message.
func (l *Logger) Debug(m string) {l.Log(DEBUG, m)}

// Debg2 logs a debug message at level 2.
func (l *Logger) Debg2(m string) {l.Log(DEBG2, m)}

// Debg3 logs a debug message at level 3.
func (l *Logger) Debg3(m string) {l.Log(DEBG3, m)}

// Debg4 logs a debug message at level 4.
func (l *Logger) Debg4(m string) {l.Log(DEBG4, m)}

// Debg5 logs a debug message at level 5.
func (l *Logger) Debg5(m string) {l.Log(DEBG5, m)}


// Notice logs a notice message using the current logger.
func Notice(m string) {Log(NOTICE, m)}

// Panic logs a panic message using the current logger and exits with code 1.
func Panic(m string) {Log(PANIC, m); _exit(1)}

// Error logs an error message using the current logger.
func Error(m string) {Log(ERROR, m)}

// Warn logs a warning message using the current logger.
func Warn(m string)  {Log(WARN,  m)}

// Info logs an info message using the current logger.
func Info(m string)  {Log(INFO,  m)}

// Debug logs a debug message using the current logger.
func Debug(m string) {Log(DEBUG, m)}

// Debg2 logs a debug message at level 2 using the current logger.
func Debg2(m string) {Log(DEBG2, m)}

// Debg3 logs a debug message at level 3 using the current logger.
func Debg3(m string) {Log(DEBG3, m)}

// Debg4 logs a debug message at level 4 using the current logger.
func Debg4(m string) {Log(DEBG4, m)}

// Debg5 logs a debug message at level 5 using the current logger.
func Debg5(m string) {Log(DEBG5, m)}

// printf like

// Noticef logs a formatted notice message.
func (l *Logger) Noticef(f string, p ...interface{}) {l.Logf(NOTICE, f, p...)}

// Panicf logs a formatted panic message and exits with code 1.
func (l *Logger) Panicf(f string, p ...interface{}) {
	l.Logf(PANIC, f, p...); _exit(1)
}

// Errorf logs a formatted error message.
func (l *Logger) Errorf(f string, p ...interface{}) {l.Logf(ERROR, f, p...)}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(f string,  p ...interface{}) {l.Logf(WARN,  f, p...)}

// Infof logs a formatted info message.
func (l *Logger) Infof(f string,  p ...interface{}) {l.Logf(INFO,  f, p...)}

// Debugf logs a formatted debug message.
func (l *Logger) Debugf(f string, p ...interface{}) {l.Logf(DEBUG, f, p...)}

// Debg2f logs a formatted debug message at level 2.
func (l *Logger) Debg2f(f string, p ...interface{}) {l.Logf(DEBG2, f, p...)}

// Debg3f logs a formatted debug message at level 3.
func (l *Logger) Debg3f(f string, p ...interface{}) {l.Logf(DEBG3, f, p...)}

// Debg4f logs a formatted debug message at level 4.
func (l *Logger) Debg4f(f string, p ...interface{}) {l.Logf(DEBG4, f, p...)}

// Debg5f logs a formatted debug message at level 5.
func (l *Logger) Debg5f(f string, p ...interface{}) {l.Logf(DEBG5, f, p...)}


// Noticef logs a formatted notice message using the current logger.
func Noticef(f string, p ...interface{}) {Logf(NOTICE, f, p...)}

// Panicf logs a formatted panic message using the current logger and
// exits with code 1.
func Panicf(f string, p ...interface{}) {Logf(PANIC, f, p...); _exit(1)}

// Errorf logs a formatted error message using the current logger.
func Errorf(f string, p ...interface{}) {Logf(ERROR, f, p...)}

// Warnf logs a formatted warning message using the current logger.
func Warnf(f string,  p ...interface{}) {Logf(WARN,  f, p...)}

// Infof logs a formatted info message using the current logger.
func Infof(f string,  p ...interface{}) {Logf(INFO,  f, p...)}

// Debugf logs a formatted debug message using the current logger.
func Debugf(f string, p ...interface{}) {Logf(DEBUG, f, p...)}

// Debg2f logs a formatted debug message at level 2 using the current logger.
func Debg2f(f string, p ...interface{}) {Logf(DEBG2, f, p...)}

// Debg3f logs a formatted debug message at level 3 using the current logger.
func Debg3f(f string, p ...interface{}) {Logf(DEBG3, f, p...)}

// Debg4f logs a formatted debug message at level 4 using the current logger.
func Debg4f(f string, p ...interface{}) {Logf(DEBG4, f, p...)}

// Debg5f logs a formatted debug message at level 5 using the current logger.
func Debg5f(f string, p ...interface{}) {Logf(DEBG5, f, p...)}

// late execution functions

// Noticel logs a message with late evaluation at notice level.
func (l *Logger) Noticel(f string, p ...interface{}) {l.Logl(NOTICE, f, p...)}

// Panicl logs a message with late evaluation at panic level and exits
// with code 1.
func (l *Logger) Panicl(f string, p ...interface{}) {
	l.Logl(PANIC, f, p...); _exit(1)
}

// Errorl logs a message with late evaluation at error level.
func (l *Logger) Errorl(f string, p ...interface{}) {l.Logl(ERROR, f, p...)}

// Warnl logs a message with late evaluation at warning level.
func (l *Logger) Warnl(f string,  p ...interface{}) {l.Logl(WARN,  f, p...)}

// Infol logs a message with late evaluation at info level.
func (l *Logger) Infol(f string,  p ...interface{}) {l.Logl(INFO,  f, p...)}

// Debugl logs a message with late evaluation at debug level.
func (l *Logger) Debugl(f string, p ...interface{}) {l.Logl(DEBUG, f, p...)}

// Debg2l logs a message with late evaluation at debug level 2.
func (l *Logger) Debg2l(f string, p ...interface{}) {l.Logl(DEBG2, f, p...)}

// Debg3l logs a message with late evaluation at debug level 3.
func (l *Logger) Debg3l(f string, p ...interface{}) {l.Logl(DEBG3, f, p...)}

// Debg4l logs a message with late evaluation at debug level 4.
func (l *Logger) Debg4l(f string, p ...interface{}) {l.Logl(DEBG4, f, p...)}

// Debg5l logs a message with late evaluation at debug level 5.
func (l *Logger) Debg5l(f string, p ...interface{}) {l.Logl(DEBG5, f, p...)}


// Noticel logs a message with late evaluation at notice level using
// the current logger.
func Noticel(f string, p ...interface{}) {Logl(NOTICE, f, p...)}

// Panicl logs a message with late evaluation at panic level using the
// current logger and exits with code 1.
func Panicl(f string, p ...interface{}) {Logl(PANIC, f, p...); _exit(1)}

// Errorl logs a message with late evaluation at error level using the
// current logger.
func Errorl(f string, p ...interface{}) {Logl(ERROR, f, p...)}

// Warnl logs a message with late evaluation at warning level using the
// current logger.
func Warnl(f string,  p ...interface{}) {Logl(WARN,  f, p...)}

// Infol logs a message with late evaluation at info level using the
// current logger.
func Infol(f string,  p ...interface{}) {Logl(INFO,  f, p...)}

// Debugl logs a message with late evaluation at debug level using the
// current logger.
func Debugl(f string, p ...interface{}) {Logl(DEBUG, f, p...)}

// Debg2l logs a message with late evaluation at debug level 2 using the
// current logger.
func Debg2l(f string, p ...interface{}) {Logl(DEBG2, f, p...)}

// Debg3l logs a message with late evaluation at debug level 3 using the
// current logger.
func Debg3l(f string, p ...interface{}) {Logl(DEBG3, f, p...)}

// Debg4l logs a message with late evaluation at debug level 4 using the
// current logger.
func Debg4l(f string, p ...interface{}) {Logl(DEBG4, f, p...)}

// Debg5l logs a message with late evaluation at debug level 5 using the
// current logger.
func Debg5l(f string, p ...interface{}) {Logl(DEBG5, f, p...)}

// Local Variables:
// tab-width: 4
// End:
