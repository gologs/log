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

	"github.com/gologs/log/context"
)

// Stream writes serialized log data ... somewhere.
type Stream interface {
	io.Writer
	// EOM should be invoked by the final "marshaler" stream op after each log message
	// has been written out. Such calls serve to frame log events.
	EOM(error) error
}

type nullStream struct{}

func (ns *nullStream) EOM(_ error) error { return nil }
func (ns *nullStream) Write(b []byte) (int, error) {
	return len(b), nil
}

var ns = &nullStream{}

// Null returns a stream that swallows all output, akin to /dev/null
func Null() Stream { return ns }

// Buffer represents a log message that may be, or has been, serialized to a Stream
type Buffer interface {
	io.WriterTo
	fmt.Stringer
	Len() int
}

// BufferedStream is a Stream implementation that buffers all writes in between calls to EOM.
type BufferedStream struct {
	bytes.Buffer
	// EOMFunc (optional) is invoked upon calls to EOM and is given the full contents of buffer.
	// References to the buffer are no longer valid upon returning from EOMFunc.
	EOMFunc func(Buffer, error) error
}

// EOM implements Stream
func (bs *BufferedStream) EOM(err error) error {
	defer bs.Reset()
	if bs.EOMFunc != nil {
		return bs.EOMFunc(&bs.Buffer, err)
	}
	return nil
}

var stdlog = &BufferedStream{
	EOMFunc: func(buf Buffer, err error) error {
		if err != nil {
			return err
		}
		// TODO(jdef) probably need to parameterize the call depth here
		return log.Output(2, buf.String())
	},
}

// SystemStream returns a buffered Stream that logs output via the standard "log" package.
func SystemStream() Stream {
	return stdlog
}

// StreamOp functions write log messages to a Stream
type StreamOp func(context.Context, Stream, string, ...interface{}) error

var nullOp = func(_ context.Context, _ Stream, _ string, _ ...interface{}) (_ error) { return }

// NullOp returns a stream op that discards all log messages, akin to /dev/null
func NullOp() StreamOp { return nullOp }

// Decorator functions typically return a StreamOp that somehow augments the functionality
// of the original StreamOp
type Decorator func(StreamOp) StreamOp

// NoDecorator returns a generator that does not modify the original StreamOp
func NoDecorator() Decorator { return func(x StreamOp) StreamOp { return x } }

// Decorators is a convenience type that make it simpler to apply multiple Decorator functions
// to a StreamOp
type Decorators []Decorator

// Decorate applies all of the decorators to the given StreamOp, in order. This means that the
// last decorator in the collection will be the first decorator invoked upon calls to the returned
// StreamOp instance.
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

// Format returns a StreamOp that uses fmt Print and Printf to format
// log writes to streams. An EOM signal is sent after every log message.
func Format(d ...Decorator) StreamOp {
	return Decorators(d).Decorate(StreamOp(
		func(_ context.Context, w Stream, m string, a ...interface{}) (err error) {
			if m != "" {
				_, err = fmt.Fprintf(w, m, a...)
			} else {
				_, err = fmt.Fprint(w, a...)
			}
			err = w.EOM(err)
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
