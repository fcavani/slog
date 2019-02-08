// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog

import (
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fcavani/e"
	"github.com/logrusorgru/aurora"
)

const (
	numTags        = 5
	fnLevelDi      = 4
	fnLevelDiPlus1 = 5
)

// Logger defines a interface to the basic logger functions.
type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})

	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Fatalln(...interface{})

	Panic(...interface{})
	Panicf(string, ...interface{})
	Panicln(...interface{})
}

// Level type represents a log level.
type Level uint8

// This constants defines the levels available to the logger.
const (
	ProtoPrio Level = iota + 1 //More priority
	DebugPrio
	InfoPrio
	ErrorPrio
	FatalPrio
	PanicPrio
	NoPrio //Less priority
)

var protoPrio = []byte("protocol")
var debugPrio = []byte("debug")
var infoPrio = []byte("info")
var errorPrio = []byte("error")
var fatalPrio = []byte("fatal")
var panicPrio = []byte("panic")
var noPriority = []byte("no priority")

// Byte returns the byte representations of a level.
func (l Level) Byte() []byte {
	switch l {
	case ProtoPrio:
		return protoPrio
	case DebugPrio:
		return debugPrio
	case InfoPrio:
		return infoPrio
	case ErrorPrio:
		return errorPrio
	case FatalPrio:
		return fatalPrio
	case PanicPrio:
		return panicPrio
	case NoPrio:
		return noPriority
	default:
		panic("this isn't a priority")
	}
}

// String returns a string representation of a level.
func (l Level) String() string {
	switch l {
	case ProtoPrio:
		return "protocol"
	case DebugPrio:
		return "debug"
	case InfoPrio:
		return "info"
	case ErrorPrio:
		return "error"
	case FatalPrio:
		return "fatal"
	case PanicPrio:
		return "panic"
	case NoPrio:
		return "no priority"
	default:
		panic("this isn't a priority")
	}
}

// Color return a function to color the message based in the log level.
func (l Level) Color(au aurora.Aurora) func(interface{}) aurora.Value {
	switch l {
	case ProtoPrio:
		return au.Cyan
	case DebugPrio:
		return au.Bold
	case InfoPrio:
		return au.Green
	case ErrorPrio:
		return au.Red
	case FatalPrio:
		return au.Magenta
	case PanicPrio:
		return nil
	case NoPrio:
		return nil
	default:
		panic("this isn't a priority")
	}
}

// ParseLevel parses the string form of a level to the type Level.
func ParseLevel(level string) (Level, error) {
	switch level {
	case "protocol", "proto":
		return ProtoPrio, nil
	case "debug":
		return DebugPrio, nil
	case "info":
		return InfoPrio, nil
	case "error":
		return ErrorPrio, nil
	case "fatal":
		return FatalPrio, nil
	case "panic":
		return PanicPrio, nil
	case "no priority":
		return NoPrio, nil
	default:
		return NoPrio, e.New("invalid priority")
	}
}

// Log is a simple log entry.
//easyjson:json
type Log struct {
	Domain    []byte
	Priority  Level
	Timestamp time.Time
	Tags      *tags
	msg       string
	DiLevel   int
	DoDi      bool
	zoneHour  int
	zoneMin   int
	zoneSig   bool
	zoneBuf   []byte
	file      string
}

// Message sets the log message.
func (l *Log) Message(str string) {
	l.msg = str
}

// Coloring color the message.
func (l *Log) Coloring(au aurora.Aurora) string {
	if au == nil {
		return l.msg
	}
	fn := l.Priority.Color(au)
	if fn == nil {
		return l.msg
	}
	return fn(l.msg).String()
}

func (l *Log) formatMessage(au aurora.Aurora) []byte {
	msg := l.Coloring(au)
	if len(msg) == 0 {
		return []byte{}
	}
	if msg[len(msg)-1] != '\n' {
		msg += "\n"
	}
	return []byte(msg)
}

// FormatMessage simple format the message without color.
func (l *Log) FormatMessage() string {
	if len(l.msg) == 0 {
		return ""
	}
	if l.msg[len(l.msg)-1] != '\n' {
		l.msg += "\n"
	}
	return l.msg
}

// String print the Log struct contents.
func (l *Log) String() string {
	return fmt.Sprintf("Domain: %v\nPriority: %v\nTimestamp: %v\nTags: %v\nMessage: %v\n",
		string(l.Domain),
		l.Priority.String(),
		l.Timestamp.Format(time.RFC3339Nano),
		l.Tags.String(),
		l.FormatMessage(),
	)
}

func (l *Log) copy() *Log {
	d := make([]byte, len(l.Domain))
	copy(d, l.Domain)
	return &Log{
		Domain:    d,
		Priority:  l.Priority,
		Timestamp: l.Timestamp,
		Tags:      l.Tags.copy(),
		msg:       l.msg,
		DiLevel:   l.DiLevel,
		DoDi:      l.DoDi,
		zoneHour:  l.zoneHour,
		zoneMin:   l.zoneMin,
		zoneSig:   l.zoneSig,
		zoneBuf:   l.zoneBuf,
	}
}

// debugInfo populates the Log struct with the debug information.
func debugInfo(level int) (file string) {
	var ok bool
	var line int
	_, file, line, ok = runtime.Caller(level)
	if ok {
		s := strings.Split(file, "/")
		length := len(s)
		if length >= 2 {
			file = strings.Join(s[length-2:length], "/") + ":" + strconv.Itoa(line)
		} else {
			file = s[0] + ":" + strconv.Itoa(line)
		}
	}
	return
}

func timeZone() (h, m int, sig bool) {
	t := time.Now()
	_, offset := t.Zone() //offset is secods East UTC
	if offset >= 0 {
		sig = true
	} else {
		offset = -offset
	}
	h = int(math.Floor(float64(offset) / 3600.0))
	m = int(math.Floor(float64(offset%3600) / 60.0))
	return
}

// SetTimeZone sets the zone info from time.Now()
func (l *Log) SetTimeZone() {
	l.zoneHour, l.zoneMin, l.zoneSig = timeZone()
	l.zoneBuf = make([]byte, 0, 6)
	if l.zoneSig {
		// TODO: check if UTC is + or -. Check if East UTC is really + before the offset.
		l.zoneBuf = append(l.zoneBuf, '+')
	} else {
		l.zoneBuf = append(l.zoneBuf, '-')
	}
	Itoa(&l.zoneBuf, l.zoneHour, 2)
	l.zoneBuf = append(l.zoneBuf, ':')
	Itoa(&l.zoneBuf, l.zoneMin, 2)
}

func (l *Log) di(deep int) {
	if l.DiLevel > 0 {
		return
	}
	l.DiLevel = deep
}

// Slog is the logger.
type Slog struct {
	// Level is the max log level that will filter the entries.
	Level Level
	// Filter filter the log entry. If return true the log entry pass the filter.
	Filter func(l *Slog) bool
	// Format a readable message from the log information
	Formatter func(l *Slog) ([]byte, error)
	// Commit sent entry to somewhere.
	Commit func(l *Slog)
	// Writer can be a destiny in the Commit function.
	Writter io.WriteCloser
	// Log entry.
	Log *Log
	// Exiter is the function called on Fatal and Panic methods.
	Exiter func(int)
	// Enable coloring of the log entry.
	colors  bool
	au      aurora.Aurora
	logPool *sync.Pool
	once    sync.Once
	Lck     *sync.Mutex
	Cp      bool
	wbuf    []byte
	wlck    sync.Mutex
}

// Itoa converts a int to a byte. i is the interger to be converted, buf is a pointer
// to the buffer that will receive the converted interger and wid is the number of
// digits, if the digits is less than wid it will be filled with zeros.
// This function come from std.
func Itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

// FormatTime is a sample function for format the data and time.
func FormatTime(buf *[]byte, t time.Time) {
	// TODO: Time zone and UTC
	year, month, day := t.Date()
	Itoa(buf, year, 4)
	*buf = append(*buf, '/')
	Itoa(buf, int(month), 2)
	*buf = append(*buf, '/')
	Itoa(buf, day, 2)
	*buf = append(*buf, ' ')
	hour, min, sec := t.Clock()
	Itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	Itoa(buf, min, 2)
	*buf = append(*buf, ':')
	Itoa(buf, sec, 2)
}

var sep = []byte(" - ")
var sepTags = []byte("- ")

// Pool of buffers to be used with formatter and commit functions.
var Pool *sync.Pool

// BufferSize is the initial allocated size of the buffers.
var BufferSize = 512

var numLogs int

func init() {
	Pool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, BufferSize)
		},
	}
}

// Init initializes the logger with domain and numLogs. numLogs is the number of
// Slog struct in the pool.
func (l *Slog) Init(domain string, nl int) error {
	numLogs = nl
	if l.Log == nil {
		l.Log = &Log{
			Priority: InfoPrio,
			Tags:     newTags(numTags),
		}
		l.Log.Domain = []byte(domain)
		if domain == "" {
			l.Log.Domain = []byte("Slog")
		}
	}

	if l.Log.Priority == 0 {
		l.Log.Priority = InfoPrio
	}
	if l.Log.Tags == nil {
		l.Log.Tags = newTags(numTags)
	}
	if len(l.Log.Domain) == 0 {
		l.Log.Domain = []byte("Slog")
	}

	l.au = aurora.NewAurora(l.colors)

	if l.Lck == nil {
		l.Lck = new(sync.Mutex)
	}

	l.Log.SetTimeZone()

	if l.Formatter == nil {
		l.Formatter = func(sl *Slog) ([]byte, error) {
			buf := Pool.Get().([]byte)
			buf = append(buf, sl.Log.Domain...)
			buf = append(buf, sep...)
			FormatTime(&buf, sl.Log.Timestamp)
			buf = append(buf, sep...)
			buf = append(buf, sl.Log.Priority.Byte()...)
			buf = append(buf, sep...)
			if len(*sl.Log.Tags) > 0 {
				buf = append(buf, []byte(sl.Log.Tags.String())...)
				buf = append(buf, sepTags...)
			}
			if sl.Log.DoDi {
				buf = append(buf, []byte(sl.Log.file)...)
				buf = append(buf, sep...)
			}
			buf = append(buf, sl.Log.formatMessage(sl.au)...)
			return buf, nil
		}
	}
	if l.Commit == nil {
		l.Commit = func(sl *Slog) {
			sl.Log.Timestamp = time.Now()
			if sl.Log.DoDi {
				sl.Log.file = debugInfo(sl.Log.DiLevel)
			}
			buf, err := sl.Formatter(sl)
			if err != nil {
				//TODO: Give to the user a nice error message.
				println("SLOG writer failed:", err)
				return
			}
			sl.Lck.Lock()
			_, err = sl.Writter.Write(buf)
			if err != nil {
				println("SLOG writer failed:", err)
			}
			Pool.Put(buf[:0])
			sl.Lck.Unlock()
		}
	}
	if l.Writter == nil {
		l.Writter = os.Stdout
	}
	if l.Exiter == nil {
		l.Exiter = os.Exit
	}
	if l.Filter == nil {
		l.Filter = func(_ *Slog) bool {
			return true
		}
	}
	if l.Level == 0 {
		l.Level = InfoPrio
	}
	l.once.Do(func() {
		l.logPool = new(sync.Pool)
		l.logPool.New = func() interface{} {
			newLog := &Log{
				Domain:   l.Log.Domain,
				Priority: InfoPrio,
				Tags:     newTags(numTags),
				DoDi:     false,
				DiLevel:  0,
			}
			newLog.SetTimeZone()
			sl := &Slog{
				Level:     l.Level,
				Formatter: l.Formatter,
				Commit:    l.Commit,
				Writter:   l.Writter,
				Exiter:    l.Exiter,
				Filter:    l.Filter,
				Log:       newLog,
				Lck:       l.Lck,
				logPool:   l.logPool,
				colors:    l.colors,
			}
			sl.au = aurora.NewAurora(sl.colors)
			return sl
		} //New
		for i := 0; i < numLogs; i++ {
			newLog := &Log{
				Domain:   l.Log.Domain,
				Priority: InfoPrio,
				Tags:     newTags(numTags),
				DoDi:     false,
				DiLevel:  0,
			}
			newLog.SetTimeZone()
			sl := &Slog{
				Level:     l.Level,
				Formatter: l.Formatter,
				Commit:    l.Commit,
				Writter:   l.Writter,
				Exiter:    l.Exiter,
				Filter:    l.Filter,
				Log:       newLog,
				Lck:       l.Lck,
				logPool:   l.logPool,
				colors:    l.colors,
			}
			sl.au = aurora.NewAurora(sl.colors)
			l.logPool.Put(sl)
		}
	})
	return nil
}

func (l *Slog) copy() *Slog {
	if l.Cp {
		return l
	}
	out := l.logPool.Get().(*Slog)
	out.Level = l.Level
	out.Log = l.Log.copy()
	out.Exiter = l.Exiter
	out.Cp = true
	out.colors = l.colors
	out.au = aurora.NewAurora(l.colors)
	return out
}

func (l *Slog) dup() *Slog {
	out := l.logPool.Get().(*Slog)
	out.Writter = l.Writter
	out.Level = l.Level
	out.Formatter = l.Formatter
	out.Commit = l.Commit
	out.Log = l.Log.copy()
	out.Exiter = l.Exiter
	out.colors = l.colors
	out.au = aurora.NewAurora(l.colors)
	return out
}

// Colors enable or disable coloring of messages in log.
func (l *Slog) Colors(b bool) {
	l.colors = b
	l.au = aurora.NewAurora(b)
}

// MakeDefault turn the behavior of actual chain of functions into default to be
// used in the next chain.
func (l *Slog) MakeDefault() *Slog {
	out := l.dup()
	out.Cp = false
	return out
}

// SetLevel set the level to filter log entries.
func (l *Slog) SetLevel(level Level) *Slog {
	l = l.copy()
	l.Level = level
	return l
}

// ProtoLevel set the log level to protocol
func (l *Slog) ProtoLevel() *Slog {
	l = l.copy()
	l.Log.Priority = ProtoPrio
	return l
}

// DebugLevel set the log level to debug
func (l *Slog) DebugLevel() *Slog {
	l = l.copy()
	l.Log.Priority = DebugPrio
	return l
}

// InfoLevel set the log level to info
func (l *Slog) InfoLevel() *Slog {
	l = l.copy()
	l.Log.Priority = InfoPrio
	return l
}

// ErrorLevel set the log level to error
func (l *Slog) ErrorLevel() *Slog {
	l = l.copy()
	l.Log.Priority = ErrorPrio
	return l
}

// Tag add tags to the log entry.
func (l *Slog) Tag(tags ...string) *Slog {
	l = l.copy()
	l.Log.Tags.Clean()
	l.Log.Tags.Add(tags...)
	return l
}

// Di add debug information to the log entry.
func (l *Slog) Di() *Slog {
	l = l.copy()
	l.Log.DoDi = true
	return l
}

func (l *Slog) di(deep int) *Slog {
	l = l.dup()
	l.Log.di(deep)
	return l
}

// NoDi disable debug info.
func (l *Slog) NoDi() *Slog {
	l = l.copy()
	l.Log.DoDi = false
	l.Log.DiLevel = 0
	return l
}

func (l *Slog) commit() {
	defer func() {
		l.Log.Priority = InfoPrio
		l.Log.DoDi = false
		l.Log.DiLevel = 0
		l.Log.Tags = newTags(numTags)
		l.Cp = false
		l.logPool.Put(l)
	}()

	// If level is less than Priority discart the log entry
	if l.Log.Priority < l.Level {
		return
	}

	if !l.Filter(l) {
		return
	}

	l.Commit(l)
}

// Print prints a log entry to the destine, this is determined by the commit
// function.
func (l *Slog) Print(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Message(fmt.Sprint(v...))
	l.commit()
}

// Printf prints a formated log entry to the destine.
func (l *Slog) Printf(s string, v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Message(fmt.Sprintf(s, v...))
	l.commit()
}

// Println prints a log entry to the destine.
func (l *Slog) Println(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Message(fmt.Sprintln(v...))
	l.commit()
}

// Error logs an error.
func (l *Slog) Error(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Priority = ErrorPrio
	l.Log.Message(fmt.Sprint(v...))
	l.commit()
}

// Errorf logs an error with format.
func (l *Slog) Errorf(s string, v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Priority = ErrorPrio
	l.Log.Message(fmt.Sprintf(s, v...))
	l.commit()
}

// Errorln logs an error.
func (l *Slog) Errorln(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Priority = ErrorPrio
	l.Log.Message(fmt.Sprintln(v...))
	l.commit()
}

// Fatal print a log entry to the destine and exit with 1.
func (l *Slog) Fatal(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Message(fmt.Sprint(v...))
	l.Log.Priority = FatalPrio
	l.commit()
	l.Writter.Close()
	l.Exiter(1)
}

// Fatalf print a formated log entry to the destine and exit with 1.
func (l *Slog) Fatalf(s string, v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Message(fmt.Sprintf(s, v...))
	l.Log.Priority = FatalPrio
	l.commit()
	l.Writter.Close()
	l.Exiter(1)
}

// Fatalln print a log entry to the destine and exit with 1.
func (l *Slog) Fatalln(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	l.Log.Message(fmt.Sprintln(v...))
	l.Log.Priority = FatalPrio
	l.commit()
	l.Writter.Close()
	l.Exiter(1)
}

// Panic print a log entry to the destine and call panic.
func (l *Slog) Panic(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	msg := fmt.Sprint(v...)
	l.Log.Message(msg)
	l.Log.Priority = PanicPrio
	l.commit()
	l.Writter.Close()
	panic(msg)
}

// Panicf print a formated log entry to the destine and call panic.
func (l *Slog) Panicf(s string, v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	msg := fmt.Sprintf(s, v...)
	l.Log.Message(msg)
	l.Log.Priority = PanicPrio
	l.commit()
	l.Writter.Close()
	panic(msg)
}

// Panicln print a log entry to the destine and call panic.
func (l *Slog) Panicln(v ...interface{}) {
	l = l.copy()
	l.Log.di(fnLevelDi)
	msg := fmt.Sprint(v...)
	l.Log.Message(msg)
	l.Log.Priority = PanicPrio
	l.commit()
	l.Writter.Close()
	panic(msg)
}

// GoPanic is use when recover from a panic and the panic must be logged
func (l *Slog) GoPanic(r interface{}, stack []byte, cont bool) {
	var msg string
	l = l.copy()
	l.Log.di(fnLevelDi)
	switch v := r.(type) {
	case string:
		msg = v + "\n"
	case fmt.Stringer:
		msg = v.String() + "\n"
	default:
		msg = fmt.Sprintln(r)
	}
	l.Log.Message(msg + "\n{" + string(stack) + "}")
	l.Log.Priority = PanicPrio
	l.commit()
	if !cont {
		l.Writter.Close()
		l.Exiter(1)
	}
}

func (l *Slog) Write(p []byte) (n int, err error) {
	l.wlck.Lock()
	defer l.wlck.Unlock()
	l.wbuf = append(l.wbuf, p...)
	for i, c := range l.wbuf {
		if c != '\n' {
			continue
		}
		tolog := l.wbuf[:i]
		l.Print(string(tolog))
		if i+1 < len(l.wbuf) {
			l.wbuf = l.wbuf[i+1:]
		} else {
			l.wbuf = l.wbuf[:0]
		}
	}
	return len(p), nil
}

// Close the logger.
func (l *Slog) Close() error {
	if l.Writter == nil {
		return nil
	}
	return e.New(l.Writter.Close())
}

var log *Slog

func init() {
	log = &Slog{
		Level: DebugPrio,
	}
	err := log.Init("", 100)
	if err != nil {
		println("SLOG: Fail to start log:", err)
		os.Exit(1)
	}
	log = log.Di().MakeDefault()
}

// DefaultLogger return the default logger. Mainly to be used with Writer interface.
func DefaultLogger() *Slog {
	return log
}

// SetOutput sets the commit out put to w.
func SetOutput(domain string, level Level, w io.WriteCloser, commiter func(sl *Slog), formatter func(l *Slog) ([]byte, error), nl int) error {
	log = &Slog{
		Commit:    commiter,
		Formatter: formatter,
		Writter:   w,
		Level:     level,
	}
	err := log.Init(domain, nl)
	if err != nil {
		return e.Forward(err)
	}
	return nil
}

// Exiter configures a function that will be called to exit the app.
func Exiter(fn func(int)) error {
	log.Exiter = fn
	err := log.Init(string(log.Log.Domain), numLogs)
	if err != nil {
		return e.Forward(err)
	}
	return nil
}

// SetLevel set the level to filter log entries.
func SetLevel(level Level) error {
	log.Level = level
	err := log.Init(string(log.Log.Domain), numLogs)
	if err != nil {
		return e.Forward(err)
	}
	return nil
}

// DebugInfo enable debug information for all messages.
func DebugInfo() {
	log = log.Di().MakeDefault()
}

// NoDebugInfo disable debug impormation for all messages.
func NoDebugInfo() {
	log = log.NoDi().MakeDefault()
}

// Di add debug information to the log message.
func Di() *Slog {
	return log.Di()
}

// NoDi disable debug information for this log entry.
func NoDi() *Slog {
	return log.NoDi()
}

// Colors enables or disables log message coloring.
func Colors(b bool) {
	log.Colors(b)
}

// Tag attach tags to the log entry
func Tag(tags ...string) *Slog {
	return log.Tag(tags...)
}

// Print prints a log entry to the destine, this is determined by the commit
// function.
func Print(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Print(vals...)
}

// Printf prints a formated log entry to the destine.
func Printf(str string, vals ...interface{}) {
	log.di(fnLevelDiPlus1).Printf(str, vals...)
}

// Println prints a log entry to the destine.
func Println(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Println(vals...)
}

// Error logs an error.
func Error(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Error(vals...)
}

// Errorf logs an error formated.
func Errorf(str string, vals ...interface{}) {
	log.di(fnLevelDiPlus1).Errorf(str, vals...)
}

// Errorln logs an error.
func Errorln(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Errorln(vals...)
}

// Fatal print a log entry to the destine and exit with 1.
func Fatal(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Fatal(vals...)
}

// Fatalf print a formated log entry to the destine and exit with 1.
func Fatalf(s string, vals ...interface{}) {
	log.di(fnLevelDiPlus1).Fatalf(s, vals...)
}

// Fatalln print a log entry to the destine and exit with 1.
func Fatalln(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Fatalln(vals...)
}

// Panic print a log entry to the destine and call panic
func Panic(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Panic(vals...)
}

// Panicf print a formated log entry to the destine and call panic.
func Panicf(s string, vals ...interface{}) {
	log.di(fnLevelDiPlus1).Panicf(s, vals...)
}

// Panicln print a log entry to the destine and call panic.
func Panicln(vals ...interface{}) {
	log.di(fnLevelDiPlus1).Panicln(vals...)
}

// ProtoLevel set the log level to protocol
func ProtoLevel() *Slog {
	return log.di(fnLevelDiPlus1).ProtoLevel()
}

// DebugLevel set the log level to debug
func DebugLevel() *Slog {
	return log.di(fnLevelDiPlus1).DebugLevel()
}

// InfoLevel set the log level to info
func InfoLevel() *Slog {
	return log.di(fnLevelDiPlus1).InfoLevel()
}

// ErrorLevel set the log level to error
func ErrorLevel() *Slog {
	return log.di(fnLevelDiPlus1).ErrorLevel()
}

// GoPanic logs a panic.
func GoPanic(r interface{}, stack []byte, cont bool) {
	log.di(fnLevelDiPlus1).GoPanic(r, stack, cont)
}

// RecoverBufferStack amount of buffer to store the stack.
var RecoverBufferStack = 4096

// Recover from panic and log the stack. If notexit is false, call l.Exiter(1),
// if not continue.
func Recover(notexit bool) {
	if r := recover(); r != nil {
		buf := make([]byte, RecoverBufferStack)
		n := runtime.Stack(buf, true)
		buf = buf[:n]
		log.di(fnLevelDiPlus1).GoPanic(r, buf, notexit)
	}
}
