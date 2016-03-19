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

// Package context is a strict subset of the interfaces declared in the canonical
// golang.org/x/net/context package.
package context

// Context is a subset of the golang.org/x/net/context Context interface
type Context interface {
	// Done returns a chan that closes to indicate termination of the calling context
	Done() <-chan struct{}
	// Value returns the value for the registered key, or else nil
	Value(key interface{}) interface{}
}

// Getter is a function that returns some context; it should never return nil.
type Getter func() Context

type nullContext int

func (c nullContext) Done() <-chan struct{}           { return nil }
func (c nullContext) Value(_ interface{}) interface{} { return nil }

// TODO exists to identify a place where better context is needed, but will be added later.
// Easy to programatically check this way.
func TODO() Context { return nullContext(0) }

// Background is a blank Context whose Done chan never closes.
func Background() Context { return nullContext(0) }

// stateful naively implements Context
type stateful struct {
	Context
	key, value interface{}
}

func (c *stateful) Value(key interface{}) interface{} {
	if key == c.key {
		return c.value
	}
	return c.Context.Value(key)
}

// WithValue returns a Context that associates value with key. Should not modify the
// original Context, `c`.
func WithValue(c Context, key, value interface{}) Context {
	return &stateful{c, key, value}
}

// Decorator functions usually return a modified version of the original Context
type Decorator func(Context) Context

// Decorators aggregates Decorator
type Decorators []Decorator

// Decorate applies all of the decorators to the given Context, in order. This means that the
// last decorator in the collection will be the first decorator invoked upon calls to the returned
// Context instance.
func (dd Decorators) Decorate(ctx Context) Context {
	for _, d := range dd {
		if d != nil {
			ctx = d(ctx)
		}
	}
	return ctx
}

// NoDecorator generates a decorator that returns the original context, unmodified
func NoDecorator() Decorator { return func(c Context) Context { return c } }

// NewDecorator generates a decorator that adds the key-value pair to a Context
func NewDecorator(key, value interface{}) Decorator {
	return func(c Context) Context {
		return WithValue(c, key, value)
	}
}

// NewGetter applies the given decorators to the supplied Getter such that the returned
// Getter generates Context objects produced by the supplied Getter and then decorated by the
// supplied decorators.
func NewGetter(ctxf Getter, d ...Decorator) Getter {
	if len(d) > 0 {
		return func() Context {
			return Decorators(d).Decorate(ctxf())
		}
	}
	return ctxf
}
