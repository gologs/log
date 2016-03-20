/*
Copyright 2016 James DeFelice

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

package ioutil

import (
	"github.com/gologs/log/context"
	"github.com/gologs/log/context/timestamp"
	"github.com/gologs/log/encoding"
	"github.com/gologs/log/levels"
)

// GlogHeader generates a stream encoding.Prefix decorator that prepends a standard glog
// header to every log message. Requires call tracking to be enabled.
func GlogHeader() encoding.Decorator {
	//TODO(jdef) obviously this isn't done yet. The point is not to emulate everything in glog or
	// any other system. The goal is to ensure that we can (somewhat) efficiently chain prefixes
	// when neeed using Iterable.
	var (
		buf = make(buffer, 20)
		sp  = []byte(" ")
	)
	return encoding.Prefix(func(c context.Context) encoding.Iterable {
		lvl := level(c)
		glogTimestamp(c, buf)
		return encoding.NewIterable(
			lvl, buf, sp,
		)
	})
}

var levelCodes = map[levels.Level][]byte{
	levels.Debug: []byte("D"),
	levels.Info:  []byte("I"),
	levels.Warn:  []byte("W"),
	levels.Error: []byte("E"),
	levels.Fatal: []byte("F"),
	levels.Panic: []byte("P"),
}

// Level generates a stream encoding.Prefix decorator that prepends a level code
// label to every log message.
func Level() encoding.Decorator {
	return encoding.Prefix(func(c context.Context) encoding.Iterable {
		return encoding.Singular(level(c))
	})
}

var unknownLevel = []byte("?")

func level(c context.Context) (result []byte) {
	result = unknownLevel
	if x, ok := levels.FromContext(c); ok {
		if code, ok := levelCodes[x]; ok {
			result = code
		}
	}
	return
}

// Timestamp generates a stream encoding.Prefix decorator that prepends a timestamp
// to every log message. The format of the timestamp is determined by the `layout` parameter.
// See time.Time.Format.
func Timestamp(layout string) encoding.Decorator {
	return encoding.Prefix(func(c context.Context) (it encoding.Iterable) {
		if ts, ok := timestamp.FromContext(c); ok {
			it = encoding.Singular([]byte(ts.Format(layout)))
		}
		return
	})
}

// String generates a stream encoding.Prefix decorator that prepends the given string to every
// log message.
func String(s string) encoding.Decorator {
	b := []byte(s)
	return encoding.Prefix(func(c context.Context) encoding.Iterable { return encoding.Singular(b) })
}

// GlogTimestamp generates a stream encoding.Prefix decorator that prepends a timestamp
// to every log message in the "glog" format.
// see https://github.com/golang/glog/
func GlogTimestamp() encoding.Decorator {
	// the formatting of this implemented was copy/pasted/hacked from the glog project
	buf := make(buffer, 20)
	return encoding.Prefix(func(c context.Context) (it encoding.Iterable) {
		if glogTimestamp(c, buf) {
			it = encoding.Singular(buf)
		}
		return
	})
}

func glogTimestamp(c context.Context, buf buffer) bool {
	ts, ok := timestamp.FromContext(c)
	if ok {
		// Avoid Fprintf, for speed. The format is so simple that we can do it quickly by hand.
		// It's worth about 3X. Fprintf is hard.
		var (
			_, month, day        = ts.Date()
			hour, minute, second = ts.Clock()
		)
		// mmdd hh:mm:ss.uuuuuu
		buf.twoDigits(0, int(month))
		buf.twoDigits(2, day)
		buf[4] = ' '
		buf.twoDigits(5, hour)
		buf[7] = ':'
		buf.twoDigits(8, minute)
		buf[10] = ':'
		buf.twoDigits(11, second)
		buf[13] = '.'
		buf.nDigits(6, 14, ts.Nanosecond()/1000, '0')
	}
	return ok
}

// buffer and related helper funcs were copied the glog project
type buffer []byte

// Some custom tiny helper functions to print the log header efficiently.

const digits = "0123456789"

// twoDigits formats a zero-prefixed two-digit integer at buf.tmp[i].
func (buf buffer) twoDigits(i, d int) {
	buf[i+1] = digits[d%10]
	d /= 10
	buf[i] = digits[d%10]
}

// nDigits formats an n-digit integer at buf.tmp[i],
// padding with pad on the left.
// It assumes d >= 0.
func (buf buffer) nDigits(n, i, d int, pad byte) {
	j := n - 1
	for ; j >= 0 && d > 0; j-- {
		buf[i+j] = digits[d%10]
		d /= 10
	}
	for ; j >= 0; j-- {
		buf[i+j] = pad
	}
}
