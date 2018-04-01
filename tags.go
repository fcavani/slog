// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog

type tags []string

func newTags(length int) *tags {
	t := make(tags, 0, length)
	return &t
}

func (t *tags) copy() *tags {
	if t == nil {
		return nil
	}
	dst := make([]string, len(*t))
	copy(dst, *t)
	tdst := tags(dst)
	return &tdst
}

func (t tags) String() (str string) {
	if t == nil {
		return ""
	}
	for i := 0; i < len(t); i++ {
		str += t[i] + " "
	}
	return
}

func (t *tags) Add(tags ...string) {
	if t == nil {
		return
	}
	*t = append(*t, tags...)
}

func (t *tags) Have(tag string) bool {
	if t == nil {
		return false
	}
	a := *t
	for _, tg := range a {
		if tg == tag {
			return true
		}
	}
	return false
}

func (t *tags) Clean() {
	if t == nil {
		return
	}
	a := *t
	a = a[:0]
	*t = a
}

func (t tags) EncodeJSON(buf *[]byte) {
	if t == nil {
		*buf = append(*buf, []byte("[]")...)
		return
	}
	*buf = append(*buf, []byte("[")...)
	l := len(t) - 1
	for i := 0; i < len(t); i++ {
		*buf = append(*buf, []byte("\"")...)
		*buf = append(*buf, []byte(t[i])...)
		*buf = append(*buf, []byte("\"")...)
		if i < l {
			*buf = append(*buf, []byte(",")...)
		}
	}
	*buf = append(*buf, []byte("]")...)
}
