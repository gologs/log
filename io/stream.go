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

	"github.com/jdef/log/context"
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

// Null returns a stream that swallows all output (like /dev/null)
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

type StreamOp func(context.Context, Stream, string, ...interface{}) error

var nullOp = func(_ context.Context, _ Stream, _ string, _ ...interface{}) (_ error) { return }

func NullOp() StreamOp { return nullOp }

type Decorator func(StreamOp) StreamOp

func NoDecorator() Decorator { return func(x StreamOp) StreamOp { return x } }

type Decorators []Decorator

func (dd Decorators) Decorate(op StreamOp) StreamOp {
	for _, d := range dd {
		if d != nil {
			op = d(op)
		}
	}
	return op
}

/*
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
func Printf(ctx context.Context, d ...Decorator) StreamOp {
	return Decorators(d).Decorate(StreamOp(
		func(ctx context.Context, w Stream, m string, a ...interface{}) (err error) {
			if len(a) > 0 && m != "" {
				_, err = fmt.Fprintf(w, m, a...)
			} else {
				_, err = fmt.Fprint(w, a...)
			}
			w.EOM(err)
			return
		}))
}

// Prefix returns a stream Decorator that outputs a prefix blob for each stream
// operation.
func Prefix(f func(context.Context) ([]byte, error)) Decorator {
	if f == nil {
		return NoDecorator()
	}
	return func(op StreamOp) StreamOp {
		return func(c context.Context, s Stream, m string, a ...interface{}) (err error) {
			var b []byte
			if b, err = f(c); err == nil && len(b) > 0 {
				_, err = s.Write(b)
			}
			if err == nil {
				err = op(c, s, m, a...)
			}
			return
		}
	}
}

// Context returns a stream Decorator that applies a context.Decorator for each
// stream operation.
func Context(f context.Decorator) Decorator {
	if f == nil {
		return NoDecorator()
	}
	return func(op StreamOp) StreamOp {
		return func(c context.Context, s Stream, m string, a ...interface{}) error {
			return op(f(c), s, m, a...)
		}
	}
}
