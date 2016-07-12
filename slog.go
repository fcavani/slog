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
)

const numTags = 5

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

// This constants defines the levels avaliable to the logger.
const (
	ProtoPrio Level = iota //More priority
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
	Message   string
	File      string
	//Pkg       string
	//Func      string
	zoneHour int
	zoneMin  int
	zoneSig  bool
	zoneBuf  []byte
}

// DebugInfo populates the Log struct with the debug informations.
func (l *Log) DebugInfo(level int) {
	if l.File != "" {
		return
	}
	var ok bool
	var line int
	//var pc uintptr
	_, l.File, line, ok = runtime.Caller(level)
	if ok {
		s := strings.Split(l.File, "/")
		length := len(s)
		if length >= 2 {
			l.File = strings.Join(s[length-2:length], "/") + ":" + strconv.Itoa(line)
		} else {
			l.File = s[0] + ":" + strconv.Itoa(line)
		}
		//f := runtime.FuncForPC(pc)
		//l.Func = f.Name()
		//i := strings.LastIndex(l.Func, ".")
		//if i > -1 {
		//	l.Pkg = l.Func[:i]
		//}
	}
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
	l.zoneBuf = make([]byte, 6)
	if l.zoneSig {
		// TODO: check if UTC is + or -. Check if East UTC is realy + before the offset.
		l.zoneBuf = append(l.zoneBuf, '+')
	} else {
		l.zoneBuf = append(l.zoneBuf, '-')
	}
	Itoa(&l.zoneBuf, l.zoneHour, 2)
	l.zoneBuf = append(l.zoneBuf, ':')
	Itoa(&l.zoneBuf, l.zoneMin, 2)
}

// Slog is the logger.
type Slog struct {
	Formatter func(l *Slog) ([]byte, error)
	Commit    func(l *Slog)
	Writter   io.WriteCloser
	Log       *Log
	logPool   *sync.Pool
	once      sync.Once
	lck       sync.Mutex
	cp        bool
}

// Itoa converts a int to a byte. i is the interger to be converted, buf is a pointer
// to the buffer that will receive the converted interge and wid is the number of
// digits, if the digits is less than wid it will be filled with zeros.
// This functoin come from std.
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
	// i < 10
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

func init() {
	Pool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, BufferSize)
		},
	}
}

// Init initializes the logger with domain and numLogs. numLogs is the number of
// Slog structs in the pool.
func (l *Slog) Init(domain string, numLogs int) error {
	l.Log = &Log{
		Priority: InfoPrio,
		Tags:     newTags(numTags),
	}
	l.Log.Domain = []byte(domain)
	if domain == "" {
		l.Log.Domain = []byte("Slog")
	}
	l.Log.SetTimeZone()

	if l.Formatter == nil {
		l.Formatter = func(l *Slog) ([]byte, error) {
			buf := Pool.Get().([]byte)
			buf = append(buf, l.Log.Domain...)
			buf = append(buf, sep...)
			//buf = append(buf, []byte(l.Log.Timestamp.Format(l.TimeLayout))...)
			FormatTime(&buf, l.Log.Timestamp)
			buf = append(buf, sep...)
			buf = append(buf, l.Log.Priority.Byte()...)
			buf = append(buf, sep...)
			buf = append(buf, []byte(l.Log.Tags.String())...)
			buf = append(buf, sepTags...)
			if l.Log.File != "" {
				buf = append(buf, []byte(l.Log.File)...)
				buf = append(buf, sep...)
			}
			buf = append(buf, []byte(l.Log.Message)...)
			return buf, nil
		}
	}
	if l.Commit == nil {
		l.Commit = func(l *Slog) {
			l.Log.Timestamp = time.Now()
			buf, err := l.Formatter(l)
			if err != nil {
				//TODO: Give to the user a nice error message.
				return
			}
			l.lck.Lock()
			_, err = l.Writter.Write(buf)
			if err != nil {
				println("SLOG writer failed:", err)
			}
			Pool.Put(buf[:0])
			l.lck.Unlock()
			l.Log.Priority = InfoPrio
			l.logPool.Put(l)
		}
	}
	if l.Writter == nil {
		l.Writter = os.Stdout
	}
	l.once.Do(func() {
		l.logPool = new(sync.Pool)
		l.logPool.New = func() interface{} {
			log := &Log{
				Domain:   l.Log.Domain,
				Priority: InfoPrio,
				Tags:     newTags(numTags),
			}
			log.SetTimeZone()
			return &Slog{
				Formatter: l.Formatter,
				Commit:    l.Commit,
				Writter:   l.Writter,
				Log:       log,
				logPool:   l.logPool,
			}
		}
		for i := 0; i < numLogs; i++ {
			log := &Log{
				Domain:   l.Log.Domain,
				Priority: InfoPrio,
				Tags:     newTags(numTags),
			}
			log.SetTimeZone()
			l.logPool.Put(&Slog{
				Formatter: l.Formatter,
				Commit:    l.Commit,
				Writter:   l.Writter,
				Log:       log,
				logPool:   l.logPool,
			})
		}
	})
	return nil
}

// copy copy only the Slog struct the Log field only the pointer is copied
func (l *Slog) copy() *Slog {
	if l.cp {
		return l
	}
	out := l.logPool.Get().(*Slog)
	out.Log = l.Log
	out.cp = true
	return out
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

// Print prints a log entry to the destinie, this is determined by the commit
// function.
func (l *Slog) Print(v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprint(v...)
	l.Commit(l)
}

// Printf prints a formated log entry to the destine.
func (l *Slog) Printf(s string, v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprintf(s, v...)
	l.Commit(l)
}

// Println prints a log entry to the destine.
func (l *Slog) Println(v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprintln(v...)
	l.Commit(l)
}

// Fatal print a log entry to the destine and exit with 1.
func (l *Slog) Fatal(v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprint(v...)
	l.Log.Priority = FatalPrio
	l.Commit(l)
	l.Writter.Close()
	os.Exit(1)
}

// Fatalf print a formated log entry to the destine and exit with 1.
func (l *Slog) Fatalf(s string, v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprintf(s, v...)
	l.Log.Priority = FatalPrio
	l.Commit(l)
	l.Writter.Close()
	os.Exit(1)
}

// Fatalln print a log entry to the destine and exit with 1.
func (l *Slog) Fatalln(v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprintln(v...)
	l.Log.Priority = FatalPrio
	l.Commit(l)
	l.Writter.Close()
	os.Exit(1)
}

// Panic print a log entry to the destine and call panic.
func (l *Slog) Panic(v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprint(v...)
	l.Log.Priority = PanicPrio
	l.Commit(l)
	l.Writter.Close()
	panic(l.Log.Message)
}

// Panicf print a formated log entry to the destine and call panic.
func (l *Slog) Panicf(s string, v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprintf(s, v...)
	l.Log.Priority = PanicPrio
	l.Commit(l)
	l.Writter.Close()
	panic(l.Log.Message)
}

// Panicln print a log entry to the destine and call panic.
func (l *Slog) Panicln(v ...interface{}) {
	l = l.copy()
	l.Log.Message = fmt.Sprintln(v...)
	l.Log.Priority = PanicPrio
	l.Commit(l)
	l.Writter.Close()
	panic(l.Log.Message)
}

// Tag add tags to the log entry.
func (l *Slog) Tag(tags ...string) *Slog {
	l.Log.Tags.Clean()
	l.Log.Tags.Add(tags...)
	return l
}

// Di add debug information to the log entry.
func (l *Slog) Di() *Slog {
	l.Log.DebugInfo(3)
	return l
}

// GoPanic is use when recover from a panic and the panic must be logged
func (l *Slog) GoPanic(r interface{}, stack []byte, cont bool) {
	l = l.copy()
	switch v := r.(type) {
	case string:
		l.Log.Message = v + "\n"
	case fmt.Stringer:
		l.Log.Message = v.String() + "\n"
	default:
		l.Log.Message = fmt.Sprintln(r)
	}
	l.Log.Message += "\n{" + string(stack) + "}"
	l.Log.Priority = PanicPrio
	l.Commit(l)
	l.Writter.Close()
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
	log = &Slog{}
	err := log.Init("", 100)
	if err != nil {
		println("SLOG: Fail to start log:", err)
		os.Exit(1)
	}
}

// SetOutput sets the commit out put to w.
func SetOutput(domain string, w io.WriteCloser, formatter func(l *Slog) ([]byte, error), numlogs int) error {
	log = &Slog{
		Formatter: formatter,
		Writter:   w,
	}
	err := log.Init(domain, numlogs)
	if err != nil {
		return e.Forward(err)
	}
	return nil
}

// Print prints a log entry to the destinie, this is determined by the commit
// function.
func Print(vals ...interface{}) {
	log.Print(vals...)
}

// Printf prints a formated log entry to the destine.
func Printf(str string, vals ...interface{}) {
	log.Printf(str, vals...)
}

// Println prints a log entry to the destine.
func Println(vals ...interface{}) {
	log.Println(vals...)
}

// Fatal print a log entry to the destine and exit with 1.
func Fatal(vals ...interface{}) {
	log.Fatal(vals...)
}

// Fatalf print a formated log entry to the destine and exit with 1.
func Fatalf(s string, vals ...interface{}) {
	log.Fatalf(s, vals...)
}

// Fatalln print a log entry to the destine and exit with 1.
func Fatalln(vals ...interface{}) {
	log.Fatalln(vals...)
}

// Panic print a log entry to the destine and call panic
func Panic(vals ...interface{}) {
	log.Panic(vals...)
}

// Panicf print a formated log entry to the destine and call panic.
func Panicf(s string, vals ...interface{}) {
	log.Panicf(s, vals...)
}

// Panicln print a log entry to the destine and call panic.
func Panicln(vals ...interface{}) {
	log.Panicln(vals...)
}

// ProtoLevel set the log level to protocol
func ProtoLevel() *Slog {
	return log.ProtoLevel()
}

// DebugLevel set the log level to debug
func DebugLevel() *Slog {
	return log.DebugLevel()
}

// InfoLevel set the log level to info
func InfoLevel() *Slog {
	return log.InfoLevel()
}

// ErrorLevel set the log level to error
func ErrorLevel() *Slog {
	return log.ErrorLevel()
}

// RecoverBufferStack amont of buffer to store the stack.
var RecoverBufferStack = 4096

// Recover from panic and log the stack. If notexit is false, call os.Exit(1),
// if not continue.
func Recover(notexit bool) {
	if r := recover(); r != nil {
		buf := make([]byte, RecoverBufferStack)
		n := runtime.Stack(buf, true)
		buf = buf[:n]
		log.GoPanic(r, buf, notexit)
	}
}
