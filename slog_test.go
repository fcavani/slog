// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog_test

import (
	golog "log"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/fcavani/e"
	. "github.com/fcavani/slog"
)

var msg = "benchmark log test"
var l = int64(len(msg))

func TestPrint(t *testing.T) {
	logger := &Slog{
		Writter: os.Stdout,
		Level:   DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger = logger.Di()

	logger.Tag("tag1", "tag2").Println(msg)
	logger.Tag("tag1", "tag2").ErrorLevel().Di().Println(msg)
	logger.Tag("tag1", "tag2").ErrorLevel().Println(msg)
	logger.Tag("tag1", "tag2").ErrorLevel().NoDi().Println(msg)

	logger = logger.DebugLevel()

	logger.Tag("tag1", "tag2").Println(msg)

}

func TestPrintJSON(t *testing.T) {
	logger := &Slog{
		Writter:   os.Stdout,
		Formatter: JSON,
		Level:     DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	logger.Tag("tag1", "tag2").ErrorLevel().Di().Println(msg)
}

func TestPrintLevel(t *testing.T) {
	logger := &Slog{
		Writter: os.Stdout,
		Level:   DebugPrio,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger.DebugLevel().Println(msg)

	// err = logger.Close()
	// if err != nil {
	// 	t.Fatal(e.Trace(e.Forward(err)))
	// }

	logger = &Slog{
		Writter: os.Stdout,
		Level:   InfoPrio,
	}
	err = logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	//logger.SetLevel(InfoPrio)

	logger.DebugLevel().Println(msg) //Não deveria aparecer

	logger.InfoLevel().Println("info")

	// err = logger.Close()
	// if err != nil {
	// 	t.Fatal(e.Trace(e.Forward(err)))
	// }

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
		b.SetBytes(l)
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
		b.SetBytes(l)
	}
}

func BenchmarkSlogNullFile(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	logger := &Slog{
		Writter: file,
		Level:   DebugPrio,
	}
	err = logger.Init("teste", 1)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Tag("tag1", "tag2").ErrorLevel().Di().Print(msg)
		b.SetBytes(l)
	}
}

func BenchmarkSlogJSONNullFile(b *testing.B) {
	file, err := os.OpenFile("/dev/null", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	logger := &Slog{
		Writter:   file,
		Formatter: JSON,
		Level:     DebugPrio,
	}
	err = logger.Init("teste", 1)
	if err != nil {
		b.Error(e.Trace(e.Forward(err)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Tag("tag1", "tag2").ErrorLevel().Di().Print(msg)
		b.SetBytes(l)
	}
}
