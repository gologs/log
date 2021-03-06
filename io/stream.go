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

// Null returns a stream that swallows all output, akin to /dev/null
func Null() Stream { return (*nullStream)(nil) }

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
	return err
}

// NewBuffered wraps the provided stream such that all generated log content is locally
// buffered before being flushed to the underlying stream upon EOM.
func NewBuffered(s Stream) Stream {
	return &BufferedStream{
		EOMFunc: func(buf Buffer, err error) error {
			if err == nil {
				_, err = buf.WriteTo(s)
			}
			return s.EOM(err)
		},
	}
}

// SystemStream returns a buffered Stream that logs output via the standard "log" package.
func SystemStream(calldepth int) Stream {
	return &BufferedStream{
		EOMFunc: func(buf Buffer, err error) error {
			if err != nil {
				return err
			}
			return log.Output(calldepth, buf.String())
		},
	}
}

type byteTracker struct {
	io.Writer
	lastByte int8
}

func (bt *byteTracker) Write(buf []byte) (int, error) {
	n, err := bt.Writer.Write(buf)
	if n > 0 {
		bt.lastByte = int8(buf[n-1])
	}
	return n, err
}

// WriterAdapter is a Stream implementation that wraps an io.Writer, passing through
// all calls to Write and invokes EOMFunc for all calls to Stream.EOM.
type WriterAdapter struct {
	io.Writer
	// EOMFunc (optional) is invoked upon calls to EOM.
	EOMFunc func(error) error
}

// EOM implements Stream, invokes EOMFunc if it's not nil.
func (wa *WriterAdapter) EOM(err error) error {
	if wa.EOMFunc != nil {
		err = wa.EOMFunc(err)
	}
	return err
}

// TextStream passes all Write calls through to the underlying Writer and upon EOM
// appends a newline if one was not included in the message.
func TextStream(w io.Writer) Stream {
	bt := &byteTracker{Writer: w}
	lf := []byte("\n")
	return &WriterAdapter{
		Writer: bt,
		EOMFunc: func(err error) error {
			if err == nil && bt.lastByte != int8(lf[0]) {
				_, err = w.Write(lf)
			}
			return err
		},
	}
}
