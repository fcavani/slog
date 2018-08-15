// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	golog "log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/fcavani/e"
	. "github.com/fcavani/slog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var msg = "benchmark log test"
var l = int64(len(msg))

type level struct {
	Level  Level
	Byte   []byte
	String string
}

var levels []level = []level{
	{ProtoPrio, []byte("protocol"), "protocol"},
	{DebugPrio, []byte("debug"), "debug"},
	{InfoPrio, []byte("info"), "info"},
	{ErrorPrio, []byte("error"), "error"},
	{FatalPrio, []byte("fatal"), "fatal"},
	{PanicPrio, []byte("panic"), "panic"},
	{NoPrio, []byte("no priority"), "no priority"},
}

func TestLevels(t *testing.T) {
	for _, l := range levels {
		if !bytes.Equal(l.Level.Byte(), l.Byte) {
			t.Fatal("Level don't match.")
		}
		if l.Level.String() != l.String {
			t.Fatal("Level don't match.")
		}
		level, err := ParseLevel(l.String)
		if err != nil {
			t.Fatal(err)
		}
		if level.String() != l.String {
			t.Fatal("Level don't match.")
		}
	}
	level, err := ParseLevel("catoto")
	if err != nil && !e.Equal(err, "invalid priority") {
		t.Fatal(err)
	} else if err == nil {
		t.Fatal("err is nil")
	}
	if level != NoPrio {
		t.Fatal("wrong level")
	}

}

func TestWrongLevelString(t *testing.T) {
	defer func() {
		r := recover()
		str, ok := r.(string)
		if !ok {
			t.Fatal("recover fail")
		}
		if str != "this isn't a priority" {
			t.Fatal("didn't fail correctely")
		}
	}()

	var wrongLevel Level = 69
	if str := wrongLevel.String(); str != "" {
		t.Fatal("Ops! It can't be right.")
	}
}

func TestWrongLevelByte(t *testing.T) {
	defer func() {
		r := recover()
		str, ok := r.(string)
		if !ok {
			t.Fatal("recover fail")
		}
		if str != "this isn't a priority" {
			t.Fatal("didn't fail correctely")
		}
	}()

	var wrongLevel Level = 69
	if b := wrongLevel.Byte(); len(b) != 0 {
		t.Fatal("Ops! It can't be right.")
	}
}

type writerCloser struct {
	*bytes.Buffer
}

func (wc *writerCloser) Close() error {
	return nil
}

func AssertLine(t *testing.T, buf *writerCloser, line string) {
	l, err := buf.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	l = l[:len(l)-1]
	s := strings.Split(l, " - ")
	if !(len(s) >= 4 && len(s) <= 6) {
		t.Fatal("split log entry failed", s)
	}
	notime := append(s[0:1], s[2:]...)
	notimeStr := strings.Join(notime, " - ")
	if notimeStr != line {
		t.Fatalf("log entry is wrong: \"%v\" != \"%v\"", notimeStr, line)
	}
}

func AssertEOF(t *testing.T, buf *writerCloser) {
	_, err := buf.ReadString('\n')
	if !e.Equal(err, io.EOF) {
		t.Fatal("expected EOF got", err)
	}
}

type jsonEntry struct {
	Domain    string
	Priority  string
	Timestamp string
	Tags      []string
	Message   string
	File      string
}

func AssertJson(t *testing.T, buf *writerCloser, v *jsonEntry) {
	val := &jsonEntry{}

	l, err := buf.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()

	err = json.Unmarshal([]byte(l), val)
	if err != nil {
		t.Fatal(err)
	}

	v.Timestamp = ""
	val.Timestamp = ""
	if !reflect.DeepEqual(val, v) {
		t.Logf("%#v\n", val)
		t.Logf("%#v\n", v)
		t.Fatal("json don't match")
	}
}

func TestPrint(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter: buf,
		Level:   DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger = logger.Di().MakeDefault()

	logger.Tag("tag1", "tag2").Println(msg)
	AssertLine(t, buf, "teste - info - tag1 tag2 - slog/slog_test.go:187 - benchmark log test")

	logger.Tag("tag1", "tag2").ErrorLevel().Di().Println(msg)
	AssertLine(t, buf, "teste - error - tag1 tag2 - slog/slog_test.go:190 - benchmark log test")

	logger.Tag("tag1", "tag2").ErrorLevel().Println(msg)
	AssertLine(t, buf, "teste - error - tag1 tag2 - slog/slog_test.go:193 - benchmark log test")

	logger.Tag("tag1", "tag2").ErrorLevel().NoDi().Println(msg)
	AssertLine(t, buf, "teste - error - tag1 tag2 - benchmark log test")

	logger = logger.DebugLevel().MakeDefault()

	logger.Tag("tag1", "tag2").Println(msg)
	AssertLine(t, buf, "teste - debug - tag1 tag2 - slog/slog_test.go:201 - benchmark log test")

}

func TestPrintJSON(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter:   buf,
		Formatter: JSON,
		Level:     DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	logger.Tag("tag1", "tag2").ErrorLevel().Di().Println(msg)
	AssertJson(t, buf, &jsonEntry{
		Domain:    "teste",
		Priority:  "error",
		Timestamp: "",
		Tags:      []string{"tag1", "tag2"},
		Message:   "benchmark log test",
		File:      "slog/slog_test.go:218",
	})
}

func TestPrintLevel(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter: buf,
		Level:   DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger.DebugLevel().Println(msg)
	AssertLine(t, buf, "teste - debug - benchmark log test")

	buf = &writerCloser{bytes.NewBuffer([]byte{})}
	logger = &Slog{
		Writter: buf,
		Level:   InfoPrio,
	}
	err = logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger.DebugLevel().Println(msg) //Não deveria aparecer
	AssertEOF(t, buf)

	logger.InfoLevel().NoDi().Println("info")
	AssertLine(t, buf, "teste - info - info")
}

func TestFreeFunc(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}
	err := SetOutput("teste", ProtoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}

	Print(msg)
	AssertLine(t, buf, "teste - info - benchmark log test")
	Printf("%v", msg)
	AssertLine(t, buf, "teste - info - benchmark log test")
	Println(msg)
	AssertLine(t, buf, "teste - info - benchmark log test")
	Error(msg)
	AssertLine(t, buf, "teste - error - benchmark log test")
	Errorf("%v", msg)
	AssertLine(t, buf, "teste - error - benchmark log test")
	Errorln(msg)
	AssertLine(t, buf, "teste - error - benchmark log test")

	Tag("tag1").Print(msg)
	AssertLine(t, buf, "teste - info - tag1 - benchmark log test")

	ProtoLevel().Print(msg)
	AssertLine(t, buf, "teste - protocol - benchmark log test")
	DebugLevel().Print(msg)
	AssertLine(t, buf, "teste - debug - benchmark log test")
	InfoLevel().Print(msg)
	AssertLine(t, buf, "teste - info - benchmark log test")
	ErrorLevel().Print(msg)
	AssertLine(t, buf, "teste - error - benchmark log test")

	Di().Print(msg)
	AssertLine(t, buf, "teste - info - slog/slog_test.go:293 - benchmark log test")
	DebugInfo()
	NoDi().Print(msg)
	AssertLine(t, buf, "teste - info - benchmark log test")
	Print(msg)
	AssertLine(t, buf, "teste - info - slog/slog_test.go:298 - benchmark log test")
}

func TestFreeFuncPanic(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}
	err := SetOutput("teste", ProtoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		r := recover()
		str, ok := r.(string)
		if !ok {
			t.Fatal("recover fail")
		}
		if str != "benchmark log test" {
			t.Fatal("didn't fail correctly:", str)
		}
		AssertLine(t, buf, "teste - panic - benchmark log test")
	}()

	Panic(msg)
}

func TestFreeFuncPanicf(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}
	err := SetOutput("teste", ProtoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		r := recover()
		str, ok := r.(string)
		if !ok {
			t.Fatal("recover fail")
		}
		if str != "benchmark log test" {
			t.Fatal("didn't fail correctly:", str)
		}
		AssertLine(t, buf, "teste - panic - benchmark log test")
	}()

	Panicf(msg)
}

func TestFreeFuncPanicln(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}
	err := SetOutput("teste", ProtoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		r := recover()
		str, ok := r.(string)
		if !ok {
			t.Fatal("recover fail")
		}
		if str != "benchmark log test" {
			t.Fatal("didn't fail correctely:", str)
		}
		AssertLine(t, buf, "teste - panic - benchmark log test")
	}()

	Panicln(msg)
}

func TestRecover(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}
	err := SetOutput("teste", ProtoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer AssertLine(t, buf, "teste - panic - panic test")
	defer Recover(true)
	panic("panic test")
}

type OddType int

func TestFreeFuncGoPanic(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}
	err := SetOutput("teste", ProtoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	GoPanic("panic test", []byte{}, true)
	AssertLine(t, buf, "teste - panic - panic test")
	GoPanic(e.New("panic test"), []byte{}, true)
	AssertLine(t, buf, "teste - panic - panic test")
	GoPanic(OddType(42), []byte{}, true)
	AssertLine(t, buf, "teste - panic - 42")
	DebugInfo()
	GoPanic("panic test", []byte{}, true)
	AssertLine(t, buf, "teste - panic - slog/slog_test.go:394 - panic test")
}

func TestCloseWriter(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter: buf,
		Level:   DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	err = logger.Close()
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
}

func TestExiter(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter: buf,
		Level:   DebugPrio,
		Exiter: func(i int) {
			return
		},
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger.Fatal(msg)
	AssertLine(t, buf, "teste - fatal - benchmark log test")
	logger.Fatalf("%v\n", msg)
	AssertLine(t, buf, "teste - fatal - benchmark log test")
	logger.Fatalln(msg)
	AssertLine(t, buf, "teste - fatal - benchmark log test")

	logger.GoPanic(msg, []byte{}, false)
	AssertLine(t, buf, "teste - panic - benchmark log test")

}

func TestFreeFuncFatal(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	err := SetOutput("teste", InfoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	Exiter(func(int) {
		return
	})

	Fatal(msg)
	AssertLine(t, buf, "teste - fatal - benchmark log test")
	Fatalf("%v", msg)
	AssertLine(t, buf, "teste - fatal - benchmark log test")
	Fatalln(msg)
	AssertLine(t, buf, "teste - fatal - benchmark log test")
}

func BenchmarkPureGolog(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	gologger := golog.New(file, "", golog.LstdFlags)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gologger.Print(msg)
	}
}

func BenchmarkLogrus(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	logrus.SetOutput(file)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logrus.Info(msg)
	}
}

const numlogs = 1

func BenchmarkSlogNullFileDi(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	logger := &Slog{
		Writter: file,
		Level:   DebugPrio,
	}
	err = logger.Init("teste", numlogs)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Tag("tag1", "tag2").ErrorLevel().Di().Print(msg)
	}
}

func BenchmarkSlogJSONNullFileDi(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	logger := &Slog{
		Writter:   file,
		Formatter: JSON,
		Level:     DebugPrio,
	}
	err = logger.Init("teste", numlogs)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Tag("tag1", "tag2").ErrorLevel().Di().Print(msg)
	}
}

func BenchmarkSlogNullFileNoDi(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	logger := &Slog{
		Writter: file,
		Level:   DebugPrio,
	}
	err = logger.Init("teste", numlogs)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Tag("tag1", "tag2").ErrorLevel().Print(msg)
	}
}

func BenchmarkSlogJSONNullFileNoDi(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	logger := &Slog{
		Writter:   file,
		Formatter: JSON,
		Level:     DebugPrio,
	}
	err = logger.Init("teste", numlogs)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Tag("tag1", "tag2").ErrorLevel().Print(msg)
	}
}

func TestSetLevel(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter: buf,
		Level:   DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger = logger.Di().MakeDefault()

	logger.Tag("tag1", "tag2").Println(msg)
	AssertLine(t, buf, "teste - info - tag1 tag2 - slog/slog_test.go:581 - benchmark log test")

	logger.Tag("tag1", "tag2").ProtoLevel().Println(msg)
	AssertEOF(t, buf)

	logger.Tag("tag1", "tag2").SetLevel(ProtoPrio).ProtoLevel().Println(msg)
	AssertLine(t, buf, "teste - protocol - tag1 tag2 - slog/slog_test.go:587 - benchmark log test")

	logger.Tag("tag1", "tag2").ProtoLevel().Println(msg)
	AssertEOF(t, buf)

	logger.Tag("tag1", "tag2").Println(msg)
	AssertLine(t, buf, "teste - info - tag1 tag2 - slog/slog_test.go:593 - benchmark log test")

	logger = logger.SetLevel(ProtoPrio).MakeDefault()

	logger.Tag("tag1", "tag2").ProtoLevel().Println(msg)
	AssertLine(t, buf, "teste - protocol - tag1 tag2 - slog/slog_test.go:598 - benchmark log test")
}

func TestFreeSetLevel(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	err := SetOutput("teste", ProtoPrio, buf, nil, nil, 100)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	ProtoLevel().Println(msg)
	AssertLine(t, buf, "teste - protocol - benchmark log test")

	err = SetLevel(InfoPrio)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	ProtoLevel().Println(msg)
	AssertEOF(t, buf)

	Println(msg)
	AssertLine(t, buf, "teste - info - benchmark log test")

}

func TestFilter(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter: buf,
		Level:   DebugPrio,
		Filter: func(sl *Slog) bool {
			return !sl.Log.Tags.Have("nolog")
		},
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger.Tag("tag1").Println(msg)
	AssertLine(t, buf, "teste - info - tag1 - benchmark log test")

	logger.Tag("nolog").Println(msg)
	AssertEOF(t, buf)

	logger.Tag("tag1").Println(msg)
	AssertLine(t, buf, "teste - info - tag1 - benchmark log test")
}

// func BenchmarkZapSugarPrintf(b *testing.B) {
// 	sugar := zap.NewExample().Sugar()
// 	defer sugar.Sync()
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		sugar.Infof("failed to fetch URL:", zap.String("url", "http://example.com"))
// 	}
// }

func BenchmarkZapLoggerPrintf(b *testing.B) {
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionConfig().EncoderConfig),
			&Discarder{},
			zap.DebugLevel,
		))
	defer logger.Sync()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("failed to fetch URL:", zap.String("url", "http://example.com"))
	}
}

// A Syncer is a spy for the Sync portion of zapcore.WriteSyncer.
type Syncer struct {
	err    error
	called bool
}

// SetError sets the error that the Sync method will return.
func (s *Syncer) SetError(err error) {
	s.err = err
}

// Sync records that it was called, then returns the user-supplied error (if
// any).
func (s *Syncer) Sync() error {
	s.called = true
	return s.err
}

// Called reports whether the Sync method was called.
func (s *Syncer) Called() bool {
	return s.called
}

// A Discarder sends all writes to ioutil.Discard.
type Discarder struct{ Syncer }

// Write implements io.Writer.
func (d *Discarder) Write(b []byte) (int, error) {
	return ioutil.Discard.Write(b)
}

func TestWriter(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter: buf,
		Level:   DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger.Write([]byte("teste "))
	logger.Write([]byte("teste\n"))
	AssertLine(t, buf, "teste - info - teste teste")
}
