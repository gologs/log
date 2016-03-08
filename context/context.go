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

package context

type Context interface {
	Done() <-chan struct{}
	Value(interface{}) interface{}
}

type nullContext <-chan struct{}

func (c nullContext) Done() <-chan struct{}           { return nil }
func (c nullContext) Value(_ interface{}) interface{} { return nil }

func None() Context { return nullContext(nil) }

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

type Decorator func(Context) Context
