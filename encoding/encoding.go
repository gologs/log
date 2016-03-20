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

package encoding

import (
	"fmt"

	"github.com/gologs/log/context"
	"github.com/gologs/log/io"
)

// Marshaler functions write log messages to a io.Stream
type Marshaler func(context.Context, io.Stream, string, ...interface{}) error

var nullMarshaler = func(_ context.Context, _ io.Stream, _ string, _ ...interface{}) (_ error) { return }

// NullMarshaler returns a stream op that discards all log messages, akin to /dev/null
func NullMarshaler() Marshaler { return nullMarshaler }

// Decorator functions typically return a Marshaler that somehow augments the functionality
// of the original Marshaler
type Decorator func(Marshaler) Marshaler

// NoDecorator returns a generator that does not modify the original Marshaler
func NoDecorator() Decorator { return func(x Marshaler) Marshaler { return x } }

// Decorators is a convenience type that make it simpler to apply multiple Decorator functions
// to a Marshaler
type Decorators []Decorator

// Decorate applies all of the decorators to the given Marshaler, in order. This means that the
// last decorator in the collection will be the first decorator invoked upon calls to the returned
// Marshaler instance.
func (dd Decorators) Decorate(op Marshaler) Marshaler {
	for _, d := range dd {
		if d != nil {
			op = d(op)
		}
	}
	return op
}

// Copy returns a deep-copy of the decorator slice; changes to the returned slice shall not
// modify in any way the original decorators slice.
func (dd Decorators) Copy() (clone Decorators) {
	if dd != nil {
		clone = make(Decorators, len(dd))
		copy(clone, dd)
	}
	return
}

// Format returns a Marshaler that uses fmt Print and Printf to format
// log writes to streams. An EOM signal is sent after every log message.
func Format(d ...Decorator) Marshaler {
	return Decorators(d).Decorate(Marshaler(
		func(_ context.Context, w io.Stream, m string, a ...interface{}) (err error) {
			if m != "" {
				_, err = fmt.Fprintf(w, m, a...)
			} else {
				_, err = fmt.Fprint(w, a...)
			}
			err = w.EOM(err)
			return
		}))
}

// Iterable generates a byte slice; returns a nil Iterable when finished.
type Iterable func() ([]byte, Iterable)

// NewIterable generates an Iterable that iterates over the given byte slices.
func NewIterable(b ...[]byte) Iterable {
	if len(b) == 0 {
		return nil
	}
	return func() ([]byte, Iterable) {
		return b[0], NewIterable(b[1:]...)
	}
}

// Singular generates an Iterable for a single buffer; more efficient if you'll only
// ever have one buffer to return from a decorator.
func Singular(b []byte) Iterable {
	if b == nil || len(b) == 0 {
		return nil
	}
	return func() ([]byte, Iterable) {
		return b, nil
	}
}

// Prefix returns a stream Decorator that outputs a prefix blob for each stream
// operation.
func Prefix(prefixf func(context.Context) Iterable) Decorator {
	if prefixf == nil {
		return NoDecorator()
	}
	return func(op Marshaler) Marshaler {
		return func(c context.Context, s io.Stream, m string, a ...interface{}) (err error) {
			var (
				it = prefixf(c)
				b  []byte
			)
			for it != nil && err == nil {
				b, it = it()
				if len(b) > 0 {
					_, err = s.Write(b)
				}
			}
			// TODO(jdef) it's not right to short circuit the marshaler here. we should be forwarding
			// the error all the way down through the ops to the final marshaler.
			if err == nil {
				err = op(c, s, m, a...)
			}
			return
		}
	}
}

// WithContext returns a stream Decorator that applies a context.Decorator for each
// stream operation.
func WithContext(f context.Decorator) Decorator {
	if f == nil {
		return NoDecorator()
	}
	return func(op Marshaler) Marshaler {
		return func(c context.Context, s io.Stream, m string, a ...interface{}) error {
			return op(f(c), s, m, a...)
		}
	}
}
