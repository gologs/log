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

package caller

import (
	"runtime"

	"github.com/gologs/log/context"
)

type (
	// Caller identifies the file, line, and function that generated a log event.
	Caller struct {
		File     string
		Line     int
		FuncName string
	}

	// Tracking enables log decorators to inject Caller information into logging Context.
	Tracking struct {
		Enabled bool
		Depth   int
	}

	key int
)

const (
	callerKey key = iota
)

// NewContext generates a Context annotated with Caller
func NewContext(ctx context.Context, file string, line int, funcName string) context.Context {
	return context.WithValue(ctx, callerKey, Caller{
		File:     file,
		Line:     line,
		FuncName: funcName,
	})
}

// FromContext extracts a Caller from the given Context
func FromContext(ctx context.Context) (Caller, bool) {
	x, ok := ctx.Value(callerKey).(Caller)
	return x, ok
}

// WithContext decorates the given context by injecting the Caller if t.Enabled is true
func WithContext(t Tracking) context.Decorator {
	if !t.Enabled {
		return context.NoDecorator()
	}
	return func(c context.Context) context.Context {
		var (
			funcName           = "???"
			pc, file, line, ok = runtime.Caller(t.Depth)
		)
		if !ok {
			file, line = "???", 0
		} else if f := runtime.FuncForPC(pc); f != nil {
			funcName = f.Name()
		}
		return NewContext(c, file, line, funcName)
	}
}
