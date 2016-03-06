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
	"fmt"
	"io"
	"log"
)

type Stream io.Writer

type StreamFunc func([]byte) (int, error)

func (f StreamFunc) Write(b []byte) (int, error) { return f(b) }

func Null() Stream {
	return StreamFunc(func(b []byte) (int, error) {
		return len(b), nil
	})
}

func StreamWriter(s Stream) io.Writer {
	return io.Writer(s)
}

func WriterStream(w io.Writer) Stream {
	return Stream(w)
}

func SystemStream() Stream {
	return StreamFunc(func(b []byte) (count int, err error) {
		err = log.Output(2, string(b))
		count += len(b)
		return
	})
}

type WriteOp func(Context, io.Writer, string, ...interface{}) error

type Decorator func(WriteOp) WriteOp

type Decorators []Decorator

func (dd Decorators) Decorate(op WriteOp) WriteOp {
	for _, d := range dd {
		op = d(op)
	}
	return op
}

type Context interface {
	Done() <-chan struct{}
}

type nullContext <-chan struct{}

func (c nullContext) Done() <-chan struct{} { return nil }

func NoContext() Context { return nullContext(nil) }

func IfElse(i bool, a, b Stream) Stream {
	if i {
		return a
	}
	return b
}

type lastByte int8

func (b *lastByte) Write(buf []byte) (int, error) {
	n := len(buf)
	if n > 0 {
		*b = lastByte(buf[n-1])
	}
	return n, nil
}

// Operator returns a WriteOp that marshals log writes to streams.
// TODO(jdef) move to logger?
func Operator(ctx Context, d ...Decorator) WriteOp {
	var (
		last         = lastByte(-1)
		needsNewline = false
		LF           = []byte{'\n'}
	)
	return Decorators(d).Decorate(WriteOp(
		func(ctx Context, w io.Writer, m string, a ...interface{}) (err error) {
			x := 0
			if needsNewline {
				x, err = w.Write(LF)
				if x > 0 {
					needsNewline = false
				}
				if err != nil {
					return
				}
			}
			n := 0
			if len(a) > 0 && m != "" {
				n, err = fmt.Fprintf(w, m, a...)
			} else {
				n, err = fmt.Fprintln(w, a...)
			}
			if err == nil && last > -1 && last != '\n' {
				x, err = w.Write(LF)
				needsNewline = x <= 0
			} else {
				needsNewline = n > 0
			}
			return
		}))
}
