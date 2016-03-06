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

package logger

import (
	stdio "io"
	"log"

	"github.com/jdef/log/io"
)

type Logger interface {
	Logf(string, ...interface{})
}

type LoggerFunc func(string, ...interface{})

func (f LoggerFunc) Logf(msg string, args ...interface{}) {
	f(msg, args...)
}

func Null() Logger { return LoggerFunc(func(_ string, _ ...interface{}) {}) }

// IgnoreErrors is a convenience func to improve readability of func invocations
// that accept an error promise.
func IgnoreErrors() chan<- error {
	return nil
}

func StreamLogger(ctx io.Context, s io.Stream, errCh chan<- error, op io.WriteOp, d ...io.Decorator) Logger {
	op = io.Decorators(d).Decorate(op)
	w := io.StreamWriter(s)
	return LoggerFunc(func(m string, a ...interface{}) {
		if err := op(ctx, w, m, a...); err != nil && errCh != nil {
			// attempt to send back errors to the caller
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	})
}

// WriterLogger generates a Logger that logs to the given Writer.
// All errors encountered while writing log messages are silently dropped.
// See io.Operator for details.
func WriterLogger(w stdio.Writer) Logger {
	var (
		ctx = io.NoContext() // TODO(jdef)
		op  = io.Operator(ctx)
	)
	return LoggerFunc(func(m string, a ...interface{}) {
		// drop errors produced here
		op(ctx, w, m, a...)
	})
}

// SystemLogger generates a Logger that logs to the golang Print family
// of functions.
func SystemLogger() Logger {
	return LoggerFunc(func(m string, a ...interface{}) {
		if len(a) > 0 {
			if m == "" {
				log.Println(a...)
			} else {
				log.Printf(m, a...)
			}
		} else {
			log.Println(m)
		}
	})
}

type Cancel interface {
	Logger
	Cancel()
}

func WithContext(ctx io.Context, logger Cancel) Logger {
	return LoggerFunc(func(msg string, args ...interface{}) {
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			logger.Logf(msg, args...)
		}()
		select {
		case <-ctx.Done():
			logger.Cancel()
			<-ch // wait for logger to return
		case <-ch:
		}

	})
}