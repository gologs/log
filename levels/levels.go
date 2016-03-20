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

package levels

import (
	"github.com/gologs/log/context"
	"github.com/gologs/log/logger"
)

// Interface is the canonical leveled logging interface.
type Interface interface {
	Debugf(string, ...interface{}) // Debugf signifies a Debug level message
	Debug(...interface{})          // Debug signifies a Debug level message, without a message format
	Infof(string, ...interface{})  // Infof signifies an Info level message
	Info(...interface{})           // Info signifies an Info level message, without a message format
	Warnf(string, ...interface{})  // Warnf signifies an Warn level message
	Warn(...interface{})           // Warn signifies an Warn level message, without a message format
	Errorf(string, ...interface{}) // Errorf signifies an Error level message
	Error(...interface{})          // Error signifies an Error level message, without a message format
	Fatalf(string, ...interface{}) // Fatalf logs and then, typically, invokes an exit func
	Fatal(...interface{})          // Fatal logs without a message format and then, typically, invokes an exit func
	Panicf(string, ...interface{}) // Panicf logs and then, typically, invokes a panic func
	Panic(...interface{})          // Panic logs without a message format and then, typically, invokes a panic func
}

// Level represents a logging priority, or threshold, usually to indicate a level
// of importance for an associated log message.
type Level int

// Debug, Info, Warn, Error, Fatal, Panic constitute the complete set of supported
// log level priorities supported by this package. Levels are bit flags which simplifies
// the task of composing a log "mask": values can simply be OR'd together.
const (
	Debug Level = 1 << iota
	Info
	Warn
	Error
	Fatal
	Panic
)

var allLevels = []Level{Debug, Info, Warn, Error, Fatal, Panic}

type key int

const (
	levelKey key = iota
)

// DecorateContext generates a context.Decorator that injects the given level into
// the Context.
func DecorateContext(lvl Level) context.Decorator {
	return func(ctx context.Context) context.Context {
		return NewContext(ctx, lvl)
	}
}

// NewContext returns a Context annotated with the given Level
func NewContext(ctx context.Context, lvl Level) context.Context {
	return context.WithValue(ctx, levelKey, lvl)
}

// FromContext attempts to extract a Level from the given Context.
func FromContext(ctx context.Context) (Level, bool) {
	x, ok := ctx.Value(levelKey).(Level)
	return x, ok
}

// this is rubbish, but it silences "go vet"s complaints about lack of format specifiers,
// and it's a dumb enough func that the golang toolchain can optimize this away
func govetIgnoreFormat() string { return "" }

type loggers struct {
	ctxf   context.Getter
	debugf logger.Logger
	infof  logger.Logger
	warnf  logger.Logger
	errorf logger.Logger
	fatalf logger.Logger
	panicf logger.Logger
}

// Debugf implements Interface
func (f *loggers) Debugf(m string, a ...interface{}) { f.debugf.Logf(f.ctxf(), m, a...) }

// Debug implements Interface
func (f *loggers) Debug(a ...interface{}) { f.debugf.Logf(f.ctxf(), govetIgnoreFormat(), a...) }

// Infof implements Interface
func (f *loggers) Infof(m string, a ...interface{}) { f.infof.Logf(f.ctxf(), m, a...) }

// Info implements Interface
func (f *loggers) Info(a ...interface{}) { f.infof.Logf(f.ctxf(), govetIgnoreFormat(), a...) }

// Warnf implements Interface
func (f *loggers) Warnf(m string, a ...interface{}) { f.warnf.Logf(f.ctxf(), m, a...) }

// Warn implements Interface
func (f *loggers) Warn(a ...interface{}) { f.warnf.Logf(f.ctxf(), govetIgnoreFormat(), a...) }

// Errorf implements Interface
func (f *loggers) Errorf(m string, a ...interface{}) { f.errorf.Logf(f.ctxf(), m, a...) }

// Error implements Interface
func (f *loggers) Error(a ...interface{}) { f.errorf.Logf(f.ctxf(), govetIgnoreFormat(), a...) }

// Fatalf implements Interface
func (f *loggers) Fatalf(m string, a ...interface{}) { f.fatalf.Logf(f.ctxf(), m, a...) }

// Fatal implements Interface
func (f *loggers) Fatal(a ...interface{}) { f.fatalf.Logf(f.ctxf(), govetIgnoreFormat(), a...) }

// Panicf implements Interface
func (f *loggers) Panicf(m string, a ...interface{}) { f.panicf.Logf(f.ctxf(), m, a...) }

// Panic implements Interface
func (f *loggers) Panic(a ...interface{}) { f.panicf.Logf(f.ctxf(), govetIgnoreFormat(), a...) }

// WithLoggers is a factory function, it generates an instance of Interface using the Logger
// instances found in the provided Indexer. If a requisite Logger is not found by the Indexer
// then all logs for that level will be silently discarded.
func WithLoggers(ctxf context.Getter, index Indexer) Interface {
	t := func(lvl Level) logger.Logger {
		logs, ok := index.Logger(lvl)
		if !ok {
			logs = logger.Null()
		}
		return logs
	}
	return &loggers{
		ctxf,
		t(Debug),
		t(Info),
		t(Warn),
		t(Error),
		t(Fatal),
		t(Panic),
	}
}

// MinThreshold generates a transform that only logs messages at or above the `min` Level.
func MinThreshold(min Level) TransformOp {
	return Accept(MatchAtOrAbove(min))
}
