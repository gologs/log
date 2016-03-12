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
	"github.com/gologs/log/io"
	"github.com/gologs/log/logger"
)

type Interface interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{}) // Fatalf logs and then invokes an exit func
	Panicf(string, ...interface{}) // Panicf logs and then invokes a panic func
}

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
	Fatal
	Panic
)

func Levels() []Level {
	return []Level{Debug, Info, Warn, Error, Fatal, Panic}
}

func (min Level) Logger(at Level, logs logger.Logger) logger.Logger {
	if at >= min {
		return logs
	}
	return logger.Null()
}

var levelCodes = map[Level][]byte{
	Debug: []byte("D"),
	Info:  []byte("I"),
	Warn:  []byte("W"),
	Error: []byte("E"),
	Fatal: []byte("F"),
	Panic: []byte("P"),
}

func Annotator() io.Decorator {
	return io.Prefix(func(c context.Context) (b []byte, err error) {
		if x, ok := FromContext(c); ok {
			if code, ok := levelCodes[x]; ok {
				b = code
			}
		}
		return
	})
}

type Transform map[Level]func(logger.Logger) logger.Logger

func (t Transform) Apply(x Level, logs logger.Logger) (Level, logger.Logger) {
	if f, ok := t[x]; ok {
		return x, f(logs)
	}
	return x, logs
}

type TransformOp func(Level, logger.Logger) (Level, logger.Logger)

type key int

const (
	levelKey key = iota
)

func (x Level) NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, levelKey, x)
}

func FromContext(ctx context.Context) (Level, bool) {
	x, ok := ctx.Value(levelKey).(Level)
	return x, ok
}

type loggers struct {
	ic     func() context.Context // initialContext
	debugf logger.Logger
	infof  logger.Logger
	warnf  logger.Logger
	errorf logger.Logger
	fatalf logger.Logger
	panicf logger.Logger
}

func (f *loggers) Debugf(m string, a ...interface{}) { f.debugf.Logf(f.ic(), m, a...) }
func (f *loggers) Infof(m string, a ...interface{})  { f.infof.Logf(f.ic(), m, a...) }
func (f *loggers) Warnf(m string, a ...interface{})  { f.warnf.Logf(f.ic(), m, a...) }
func (f *loggers) Errorf(m string, a ...interface{}) { f.errorf.Logf(f.ic(), m, a...) }
func (f *loggers) Fatalf(m string, a ...interface{}) { f.fatalf.Logf(f.ic(), m, a...) }
func (f *loggers) Panicf(m string, a ...interface{}) { f.panicf.Logf(f.ic(), m, a...) }

func WithLoggers(ctx context.Context, debugf, infof, warnf, errorf, fatalf, panicf logger.Logger) Interface {
	check := func(x logger.Logger) logger.Logger {
		if x == nil {
			return logger.Null()
		}
		return x
	}
	return &loggers{
		func() context.Context { return ctx },
		check(debugf),
		check(infof),
		check(warnf),
		check(errorf),
		check(fatalf),
		check(panicf),
	}
}

func (min Level) Min() TransformOp {
	return func(x Level, logs logger.Logger) (Level, logger.Logger) {
		return x, min.Logger(x, logs)
	}
}
