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

	"github.com/jdef/log/context"
	"github.com/jdef/log/logger"
)

type (
	Caller struct {
		File string
		Line int
	}

	key int
)

const (
	callerKey key = iota
)

func NewContext(ctx context.Context, file string, line int) context.Context {
	return context.WithValue(ctx, callerKey, Caller{File: file, Line: line})
}

func FromContext(ctx context.Context) (Caller, bool) {
	x, ok := ctx.Value(callerKey).(Caller)
	return x, ok
}

func Logger(calldepth int, logs logger.Logger) logger.Logger {
	return logger.LoggerFunc(func(c context.Context, msg string, args ...interface{}) {
		_, file, line, ok := runtime.Caller(calldepth)
		if !ok {
			file, line = "???", 0
		}
		logs.Logf(NewContext(c, file, line), msg, args...)
	})
}
