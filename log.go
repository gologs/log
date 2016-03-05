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

package log

import (
	"log"
	"os"
)

var (
	// Exit is invoked when a log function requires the program to forcibly terminate.
	// This behavior is encapsulated in a variable so that users of this package can
	// customize the exit behavior. For example, it is tedious to unit test a program
	// that may invoke Fatalf if that call results in a call to `os.Exit`. Test authors
	// may find it convenient to override the value of this func simplify such tests.
	// Exit expects to be invoked by calls to Fatalf.
	Exit = func(code int) { os.Exit(code) }

	// Panic invokes `panic(msg)` by default. It's declared as a variable here to allow
	// for customization. Expected to be invoked by calls to Panicf.
	Panic = func(msg string, _ ...interface{}) { panic(msg) }
)

type Logger interface {
	Logf(string, ...interface{})
}

type LogFunc func(string, ...interface{})

func (f LogFunc) Logf(msg string, args ...interface{}) {
	f(msg, args...)
}

func DevNull() Logger { return Logger(LogFunc(func(_ string, _ ...interface{}) {})) }

type Interface interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{}) // Fatalf logs and then invokes Exit(1)
	Panicf(string, ...interface{}) // Panicf logs and then invokes Panic(msg,args...)
}

type funcs struct {
	debugf Logger
	infof  Logger
	warnf  Logger
	errorf Logger
	fatalf Logger
	panicf Logger
}

func (f *funcs) Debugf(msg string, args ...interface{}) { f.debugf.Logf(msg, args...) }
func (f *funcs) Infof(msg string, args ...interface{})  { f.infof.Logf(msg, args...) }
func (f *funcs) Warnf(msg string, args ...interface{})  { f.warnf.Logf(msg, args...) }
func (f *funcs) Errorf(msg string, args ...interface{}) { f.errorf.Logf(msg, args...) }
func (f *funcs) Fatalf(msg string, args ...interface{}) {
	defer Exit(1)
	f.fatalf.Logf(msg, args...)
}
func (f *funcs) Panicf(msg string, args ...interface{}) {
	defer Panic(msg, args...)
	f.panicf.Logf(msg, args...)
}

func Levels(debugf, infof, warnf, errorf, fatalf, panicf Logger) Interface {
	check := func(x Logger) Logger {
		if x == nil {
			return DevNull()
		}
		return x
	}
	return &funcs{check(debugf), check(infof), check(warnf), check(errorf), check(fatalf), check(panicf)}
}

func If(i bool, a, b Logger) Logger {
	if i {
		return a
	}
	return b
}

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

func (min Level) LogAt(at Level, logger Logger) Logger {
	return If(at >= min, logger, DevNull())
}

func Threshold(min Level, logger Logger) Interface {
	return Levels(
		min.LogAt(LevelDebug, logger),
		min.LogAt(LevelInfo, logger),
		min.LogAt(LevelWarn, logger),
		min.LogAt(LevelError, logger),
		min.LogAt(LevelFatal, logger),
		min.LogAt(LevelPanic, logger),
	)
}

type Context interface {
	Done() <-chan struct{}
}

type CancelLogger interface {
	Logger
	Cancel()
}

func WithContext(ctx Context, logger CancelLogger) Logger {
	return LogFunc(func(msg string, args ...interface{}) {
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

func System() Logger {
	return LogFunc(func(m string, a ...interface{}) {
		if len(a) > 0 {
			if m == "" {
				log.Print(a...)
			} else {
				log.Printf(m, a...)
			}
		} else {
			log.Println(m)
		}
	})
}
