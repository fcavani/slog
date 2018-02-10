// Copyright 2016 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog

type tags []string

func newTags(legth int) *tags {
	t := make(tags, 0, legth)
	return &t
}

func (t *tags) copy() *tags {
	dst := make([]string, len(*t))
	copy(dst, *t)
	tdst := tags(dst)
	return &tdst
}

func (t tags) String() (str string) {
	for i := 0; i < len(t); i++ {
		str += t[i] + " "
	}
	return
}

func (t *tags) Add(tags ...string) {
	*t = append(*t, tags...)
}

func (t *tags) Clean() {
	a := *t
	a = a[:0]
	*t = a
}

func (t tags) EncodeJSON(buf *[]byte) {
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
