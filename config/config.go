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

package config

import (
	"log"
	"os"
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
func (f *funcs) Fatalf(msg string, args ...interface{}) { f.fatalf.Logf(msg, args...) }
func (f *funcs) Panicf(msg string, args ...interface{}) { f.panicf.Logf(msg, args...) }

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

// TODO(jdef) figure out a way to incorporate this seamlessly
func (x Level) Annotated(logger Logger) Logger {
	codes := map[Level]string{
		LevelDebug: "D",
		LevelInfo:  "I",
		LevelWarn:  "W",
		LevelError: "E",
		LevelFatal: "F",
		LevelPanic: "P",
	}
	return LogFunc(func(m string, a ...interface{}) {
		logger.Logf(codes[x]+m, a...)
	})
}

func Threshold(min Level, logger Logger, fexit func(int), fpanic func(string)) Interface {
	return Levels(
		min.LogAt(LevelDebug, logger),
		min.LogAt(LevelInfo, logger),
		min.LogAt(LevelWarn, logger),
		min.LogAt(LevelError, logger),
		LogFunc(func(m string, a ...interface{}) {
			defer fexit(1)
			min.LogAt(LevelFatal, logger).Logf(m, a...)
		}),
		LogFunc(func(m string, a ...interface{}) {
			defer fpanic(m)
			min.LogAt(LevelPanic, logger).Logf(m, a...)
		}),
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

type Config struct {
	Level Level
	Sink  Logger
	Exit  func(int)
	Panic func(string)
}

var (
	// DefaultExit is the default exit function; an exit function is invoked when a
	// log invocation requires the program to forcibly terminate.
	defaultExit = func(code int) { os.Exit(code) }

	DefaultConfig = Config{
		Level: LevelInfo,                   // Level defaults to LevelInfo
		Sink:  System(),                    // Sink defaults to System()
		Exit:  defaultExit,                 // Exit defaults to invoking os.Exit
		Panic: func(m string) { panic(m) }, // Panic default to invoking panic(msg)
	}

	// Default logs everything "info" and higher ("warn", "error", ...) to DefaultSink
	Default = func() (i Interface) { i, _ = DefaultConfig.With(NoOption()); return }()
)

// Option is a functional option interface for making changes to a Config
type Option func(*Config) Option

// NoOption returns an option that doesn't make any changes to a Config.
// Use to improve readability or as a noop default Option.
func NoOption() (opt Option) {
	opt = Option(func(_ *Config) Option { return opt })
	return
}

func (cfg Config) With(opt ...Option) (Interface, Option) {
	lastOpt := NoOption()
	for _, o := range opt {
		lastOpt = o(&cfg)
	}
	return Threshold(cfg.Level, cfg.Sink, cfg.Exit, cfg.Panic), lastOpt
}

func (level Level) Option() Option {
	return func(c *Config) Option {
		old := c.Level
		c.Level = level
		return old.Option()
	}
}

func Sink(logger Logger) Option {
	return func(c *Config) Option {
		old := c.Sink
		c.Sink = logger
		return Sink(old)
	}
}

func Exit(f func(int)) Option {
	return func(c *Config) Option {
		old := c.Exit
		c.Exit = f
		return Exit(old)
	}
}

func Panic(f func(msg string)) Option {
	return func(c *Config) Option {
		old := c.Panic
		c.Panic = f
		return Panic(old)
	}
}
