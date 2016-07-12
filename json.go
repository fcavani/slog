// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog

import "time"

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
	// _, offset := t.Zone() //offset is secods East UTC
	// if offset < 0 {
	// 	offset = -offset
	// 	*buf = append(*buf, '-')
	// } else {
	// 	// TODO: check if UTC is + or -. Check if East UTC is realy + before the offset.
	// 	*buf = append(*buf, '+')
	// }
	// h := int(math.Floor(float64(offset) / 3600.0))
	// s := int(math.Floor(float64(offset%3600) / 60.0))
	// Itoa(buf, h, 2)
	// *buf = append(*buf, ':')
	// Itoa(buf, s, 2)
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
	//2016-03-24T21:26:37.556839791-03:00
	//["tag1","tag2"]
	buf := Pool.Get().([]byte)
	//JSONTemplate := `{"Domain":%v,"Priority":%v,"Timestamp":"%v","Tags":%d,"Message":"%d","File":"%d"}` + "\n"
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
	buf = append(buf, []byte(l.Log.Message)...)
	buf = append(buf, file...)
	if l.Log.File != "" {
		buf = append(buf, []byte(l.Log.File)...)
	}
	buf = append(buf, closeing...)
	return buf, nil
}
