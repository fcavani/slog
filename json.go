// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog

import (
	"strings"
	"time"
)

func formatJSONTime(buf *[]byte, t time.Time) {
	year, month, day := t.Date()
	Itoa(buf, year, 4)
	*buf = append(*buf, '-')
	Itoa(buf, int(month), 2)
	*buf = append(*buf, '-')
	Itoa(buf, day, 2)
	*buf = append(*buf, 'T')
	hour, min, sec := t.Clock()
	Itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	Itoa(buf, min, 2)
	*buf = append(*buf, ':')
	Itoa(buf, sec, 2)
	nano := t.Nanosecond()
	*buf = append(*buf, '.')
	Itoa(buf, nano, 9)
}

var domain = []byte("{\"Domain\":\"")
var prio = []byte("\",\"Priority\":\"")
var ts = []byte("\",\"Timestamp\":\"")
var tgs = []byte("\",\"Tags\":")
var msg = []byte(",\"Message\":\"")
var file = []byte("\",\"File\":\"")
var closeing = []byte("\"}\n")

// JSON convert a log entry to json.
func JSON(l *Slog) ([]byte, error) {
	m := strings.Replace(l.Log.FormatMessage(), "\n", " ", -1)
	if len(m) > 0 && m[len(m)-1] == ' ' {
		m = m[:len(m)-1]
	}
	buf := Pool.Get().([]byte)
	buf = append(buf, domain...)
	buf = append(buf, l.Log.Domain...)
	buf = append(buf, prio...)
	buf = append(buf, l.Log.Priority.Byte()...)
	buf = append(buf, ts...)
	formatJSONTime(&buf, l.Log.Timestamp)
	buf = append(buf, l.Log.zoneBuf...)
	buf = append(buf, tgs...)
	l.Log.Tags.EncodeJSON(&buf)
	buf = append(buf, msg...)
	buf = append(buf, []byte(m)...)
	buf = append(buf, file...)
	if l.Log.DoDi {
		buf = append(buf, []byte(l.Log.file)...)
	}
	buf = append(buf, closeing...)
	return buf, nil
}
