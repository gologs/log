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
	"github.com/jdef/log/context"
	"github.com/jdef/log/io"
	"github.com/jdef/log/logger"
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
	return func(op io.StreamOp) io.StreamOp {
		return func(c context.Context, s io.Stream, m string, a ...interface{}) (err error) {
			if x, ok := FromContext(c); ok {
				if code, ok := levelCodes[x]; ok {
					_, err = s.Write(code)
				}
			}
			if err == nil {
				err = op(c, s, m, a...)
			}
			return
		}
	}
}

type Transform map[Level]func(logger.Logger) logger.Logger

func (t Transform) Apply(x Level, logs logger.Logger) (Level, logger.Logger) {
	if f, ok := t[x]; ok {
		return x, f(logs)
	}
	return x, logs
}

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
	debugf logger.Logger
	infof  logger.Logger
	warnf  logger.Logger
	errorf logger.Logger
	fatalf logger.Logger
	panicf logger.Logger
}

func (f *loggers) Debugf(msg string, args ...interface{}) { f.debugf.Logf(msg, args...) }
func (f *loggers) Infof(msg string, args ...interface{})  { f.infof.Logf(msg, args...) }
func (f *loggers) Warnf(msg string, args ...interface{})  { f.warnf.Logf(msg, args...) }
func (f *loggers) Errorf(msg string, args ...interface{}) { f.errorf.Logf(msg, args...) }
func (f *loggers) Fatalf(msg string, args ...interface{}) { f.fatalf.Logf(msg, args...) }
func (f *loggers) Panicf(msg string, args ...interface{}) { f.panicf.Logf(msg, args...) }

func WithLoggers(debugf, infof, warnf, errorf, fatalf, panicf logger.Logger) Interface {
	check := func(x logger.Logger) logger.Logger {
		if x == nil {
			return logger.Null()
		}
		return x
	}
	return &loggers{
		check(debugf),
		check(infof),
		check(warnf),
		check(errorf),
		check(fatalf),
		check(panicf),
	}
}
