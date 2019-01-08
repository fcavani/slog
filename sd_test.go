// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog_test

import (
	"bytes"
	"testing"

	"github.com/fcavani/e"
	. "github.com/fcavani/slog"
)

func TestSdPrint(t *testing.T) {
	buf := &writerCloser{bytes.NewBuffer([]byte{})}

	logger := &Slog{
		Writter:   buf,
		Level:     ProtoPrio,
		Commit:    CommitSd,
		Formatter: SdFormater,
	}
	err := logger.Init("teste", 1)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}

	logger = logger.Di().MakeDefault()

	logger.Tag("tag1", "tag2").Println(msg)
	AssertLine(t, buf, "teste - info - tag1 tag2 - slog/sd_test.go:31 - benchmark log test")

	Testing(true)

	logger.ProtoLevel().Tag("tag1", "tag2").Println(msg)
	logger.DebugLevel().Tag("tag1", "tag2").Println(msg)
	logger.InfoLevel().Tag("tag1", "tag2").Println(msg)
	logger.ErrorLevel().Tag("tag1", "tag2").Println(msg)
}
