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

package io

import (
	"bytes"
	"fmt"
	"io"
	"log"
)

type Stream interface {
	io.Writer
	EOM(error)
}

type nullStream struct{}

func (ns *nullStream) EOM(_ error) {}
func (ns *nullStream) Write(b []byte) (int, error) {
	return len(b), nil
}

var ns = &nullStream{}

func Null() Stream { return ns }

type BufferedStream struct {
	bytes.Buffer
	EOMFunc func(*bytes.Buffer, error)
}

func (bs *BufferedStream) EOM(err error) {
	defer bs.Reset()
	if bs.EOMFunc != nil {
		bs.EOMFunc(&bs.Buffer, err)
	}
}

var stdlog = &BufferedStream{
	EOMFunc: func(buf *bytes.Buffer, _ error) {
		// ignore errors
		log.Output(2, buf.String())
	},
}

func SystemStream() Stream {
	return stdlog
}

type StreamOp func(Context, Stream, string, ...interface{}) error

var nullOp = func(_ Context, _ Stream, _ string, _ ...interface{}) (_ error) { return }

func NullOp() StreamOp { return nullOp }

type Decorator func(StreamOp) StreamOp

type Decorators []Decorator

func (dd Decorators) Decorate(op StreamOp) StreamOp {
	for _, d := range dd {
		if d != nil {
			op = d(op)
		}
	}
	return op
}

type Context interface {
	Done() <-chan struct{}
}

type nullContext <-chan struct{}

func (c nullContext) Done() <-chan struct{} { return nil }

func NoContext() Context { return nullContext(nil) }

/*
func IfElse(i bool, a, b Stream) Stream {
	if i {
		return a
	}
	return b
}

type byteTracker struct {
	Stream
	lastByte int8
}

func (bt *byteTracker) Write(buf []byte) (int, error) {
	n, err := bt.Stream.Write(buf)
	if n > 0 {
		bt.lastByte = int8(buf[n-1])
	}
	return n, err
}
*/

// Printf returns a StreamOp that uses fmt Print and Printf to format
// log writes to streams. An EOM signal is sent after every log message.
func Printf(ctx Context, d ...Decorator) StreamOp {
	return Decorators(d).Decorate(StreamOp(
		func(ctx Context, w Stream, m string, a ...interface{}) (err error) {
			if len(a) > 0 && m != "" {
				_, err = fmt.Fprintf(w, m, a...)
			} else {
				_, err = fmt.Fprint(w, a...)
			}
			w.EOM(err)
			return
		}))
}
