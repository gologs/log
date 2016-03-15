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
	Infof(string, ...interface{})  // Infof signifies an Info level message
	Warnf(string, ...interface{})  // Warnf signifies an Warn level message
	Errorf(string, ...interface{}) // Errorf signifies an Error level message
	Fatalf(string, ...interface{}) // Fatalf logs and then, typically, invokes an exit func
	Panicf(string, ...interface{}) // Panicf logs and then, typically, invokes a panic func
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

// Transform collects Decorators that are applied to `Logger`s for specific `Level`s.
type Transform map[Level]logger.Decorator

// Apply decorates the given Logger using the Decorator as specified for the given
// Level (via the receiving Transform)
func (t Transform) Apply(x Level, logs logger.Logger) (Level, logger.Logger) {
	if f, ok := t[x]; ok {
		return x, f(logs)
	}
	return x, logs
}

// TransformOp typically returns the same Level with a modified Logger
type TransformOp func(Level, logger.Logger) (Level, logger.Logger)

// TransformOps aggregates TransformOp and allows such operators to be applied in bulk
type TransformOps []TransformOp

// Apply executes all the TransformOps against the input, first to last, and returns the result
func (ops TransformOps) Apply(x Level, logs logger.Logger) (Level, logger.Logger) {
	for _, t := range ops {
		if t != nil {
			x, logs = t(x, logs)
		}
	}
	return x, logs
}

// Copy returns a clone of the ops slice that's independent of the original
func (ops TransformOps) Copy() TransformOps {
	if ops == nil {
		return ops
	}
	clone := make(TransformOps, len(ops))
	copy(clone, ops)
	return clone
}

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

// Infof implements Interface
func (f *loggers) Infof(m string, a ...interface{}) { f.infof.Logf(f.ctxf(), m, a...) }

// Warnf implements Interface
func (f *loggers) Warnf(m string, a ...interface{}) { f.warnf.Logf(f.ctxf(), m, a...) }

// Errorf implements Interface
func (f *loggers) Errorf(m string, a ...interface{}) { f.errorf.Logf(f.ctxf(), m, a...) }

// Fatalf implements Interface
func (f *loggers) Fatalf(m string, a ...interface{}) { f.fatalf.Logf(f.ctxf(), m, a...) }

// Panicf implements Interface
func (f *loggers) Panicf(m string, a ...interface{}) { f.panicf.Logf(f.ctxf(), m, a...) }

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

// MinTransform generates a transform that only logs messages at or above the `min` Level.
func MinTransform(min Level) TransformOp {
	return Accept(MatchAtOrAbove(min))
}

// Indexer functions map a Level to a Logger, or else return false
type Indexer interface {
	Logger(Level) (logger.Logger, bool)
}

// IndexerFunc is the functional adaptation of the Indexer interface
type IndexerFunc func(Level) (logger.Logger, bool)

// Logger implements Indexer
func (f IndexerFunc) Logger(lvl Level) (logger.Logger, bool) { return f(lvl) }

type levelMap map[Level]logger.Logger

func (lm levelMap) Logger(lvl Level) (logs logger.Logger, ok bool) {
	logs, ok = lm[lvl]
	return
}

// NewIndexer builds a logger for each Level, starting with the original Logger
// in the given Indexer and then applying the provided transforms. If nil is given
// for `levels` then all log levels are assumed.
func NewIndexer(idx Indexer, levels []Level, chain ...TransformOp) Indexer {
	if levels == nil {
		levels = allLevels
	}
	m := make(levelMap, len(levels))
	for _, x := range levels {
		logs, ok := idx.Logger(x)
		if !ok {
			continue
		}
		x, logs = TransformOps(chain).Apply(x, logs)
		m[x] = logs
	}
	return m
}

// Filter returns true if the given level is accepted
type Filter func(Level) bool

// MatchAny filters return true if the logical AND of a level with the given levelMask is non-zero
func MatchAny(levelMask Level) Filter { return func(x Level) bool { return (levelMask & x) != 0 } }

// MatchExact filters return true if a tested level is identical to the level provided to the matcher.
func MatchExact(lvl Level) Filter { return func(x Level) bool { return x == lvl } }

// MatchAtOrAbove filters return true if the tested level is the same or higher then that provided
// to the matcher.
func MatchAtOrAbove(lvl Level) Filter { return func(x Level) bool { return x >= lvl } }

// Broadcast replicates log messages for the accepted levels to all the provided loggers.
// If replace is false, a copy of the log message is also sent to the original input logger
// of the returned TransformOp. If replace is true and len(log) == 0 then accepted logs
// events will be dropped (sent to logger.Null()). Log messages that are not accepted by
// the filter are simply passed through the original logger.
func Broadcast(filter Filter, replace bool, log ...logger.Logger) TransformOp {
	return func(x Level, ll logger.Logger) (Level, logger.Logger) {
		if filter(x) {
			if replace {
				if len(log) == 0 {
					return x, logger.Null()
				}
				return x, logger.Multi(log...)
			}
			if len(log) == 0 {
				return x, ll // edge case, but there's no use wrapping here
			}
			return x, logger.Multi(append(log, ll)...)
		}
		return x, ll
	}
}

// Accept drops log messages whose log level does not match the given filter.
func Accept(filter Filter) TransformOp {
	return func(x Level, ll logger.Logger) (Level, logger.Logger) {
		if filter(x) {
			return x, ll
		}
		return x, logger.Null()
	}
}
