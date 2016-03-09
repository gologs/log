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
	"log"

	"github.com/jdef/log/context"
	"github.com/jdef/log/io"
)

type Logger interface {
	Logf(context.Context, string, ...interface{})
}

type Func func(context.Context, string, ...interface{})

func (f Func) Logf(c context.Context, msg string, args ...interface{}) {
	f(c, msg, args...)
}

func Null() Logger { return Func(func(_ context.Context, _ string, _ ...interface{}) {}) }

// Tee returns a Logger that copies log events to both `logger` and `dup`.
func Tee(logger, dup Logger) Logger {
	return Func(func(c context.Context, m string, a ...interface{}) {
		logger.Logf(c, m, a...)
		dup.Logf(c, m, a...)
	})
}

// IgnoreErrors is a convenience func to improve readability of func invocations
// that accept an error promise.
func IgnoreErrors() chan<- error {
	return nil
}

// WithStream generates a Logger that writes log events to the given
// io.Stream using the given `op` marshaler. It is expected that a marshaler
// will invoke EOM after processing each log event.
func WithStream(s io.Stream, op io.StreamOp, errCh chan<- error) Logger {
	return Func(func(ctx context.Context, m string, a ...interface{}) {
		if err := op(ctx, s, m, a...); err != nil && errCh != nil {
			// attempt to send back errors to the caller
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	})
}

type Decorator func(Logger) Logger

func NoDecorator() Decorator { return func(x Logger) Logger { return x } }

func Context(d context.Decorator) Decorator {
	if d == nil {
		return NoDecorator()
	}
	return func(logger Logger) Logger {
		return Func(func(c context.Context, m string, a ...interface{}) {
			logger.Logf(d(c), m, a...)
		})
	}
}

/*
type ignoreEOM struct {
	stdio.Writer
}

func (i *ignoreEOM) EOM(_ error) {}

// WriterLogger generates a Logger that logs to the given Writer.
// All errors encountered while writing log messages are silently dropped.
// See io.Operator for details.
func WriterLogger(w stdio.Writer) Logger {
	var (
		ctx = io.NoContext() // TODO(jdef)
		op  = io.Printf(ctx)
		// TODO(jdef) should better handle EOM's by checking for LF
	)
	s := &ignoreEOM{w}
	return Func(func(m string, a ...interface{}) {
		// drop errors produced here
		op(ctx, s, m, a...)
	})
}
*/

// SystemLogger generates a Logger that logs to the golang Print family
// of functions.
func SystemLogger() Logger {
	return Func(func(_ context.Context, m string, a ...interface{}) {
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

/*
type Cancel interface {
	Logger
	Cancel()
}

func WithContext(ctx context.Context, logger Cancel) Logger {
	return Func(func(msg string, args ...interface{}) {
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
*/
