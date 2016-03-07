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
	"os"

	"github.com/jdef/log/io"
	"github.com/jdef/log/levels"
	"github.com/jdef/log/logger"
)

// ExitCode is passed to exit functions that are invoked upon calls to Fatalf
var ExitCode = 1

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

func WithLevelLoggers(debugf, infof, warnf, errorf, fatalf, panicf logger.Logger) levels.Interface {
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

func LeveledStreamer(
	ctx io.Context,
	min levels.Level,
	s io.Stream,
	marshaler io.StreamOp,
	t levels.Transform,
	decorators ...io.Decorator,
) levels.Interface {
	if marshaler == nil {
		marshaler = io.Printf(ctx)
	}
	if s == nil {
		s = io.SystemStream()
	}

	applyAnnotations := false
	if len(decorators) == 0 {
		applyAnnotations = true
	}

	// TODO(jdef) thinking about adding name/value pair support to Context so that I
	// can embed the log level there. That way the annotator decorator isn't so special
	// cased here.

	logAt := func(level levels.Level, d ...io.Decorator) (levels.Level, logger.Logger) {
		var annotator io.Decorator
		if applyAnnotations {
			annotator = level.Annotated()
		}
		d = append(d, annotator)
		return level, logger.StreamLogger(ctx, s, logger.IgnoreErrors(), marshaler, d...)
	}
	return WithLevelLoggers(
		min.Logger(t.Apply(logAt(levels.Debug, decorators...))),
		min.Logger(t.Apply(logAt(levels.Info, decorators...))),
		min.Logger(t.Apply(logAt(levels.Warn, decorators...))),
		min.Logger(t.Apply(logAt(levels.Error, decorators...))),
		min.Logger(t.Apply(logAt(levels.Fatal, decorators...))),
		min.Logger(t.Apply(logAt(levels.Panic, decorators...))),
	)
}

func LeveledLogger(min levels.Level, logs logger.Logger, t levels.Transform) levels.Interface {
	if logs == nil {
		logs = logger.SystemLogger()
	}

	// TODO(jdef) need to make this thread safe

	return WithLevelLoggers(
		min.Logger(t.Apply(levels.Debug, logs)),
		min.Logger(t.Apply(levels.Info, logs)),
		min.Logger(t.Apply(levels.Warn, logs)),
		min.Logger(t.Apply(levels.Error, logs)),
		min.Logger(t.Apply(levels.Fatal, logs)),
		min.Logger(t.Apply(levels.Panic, logs)),
	)
}

func safeExit(fexit func(int)) func(int) {
	if fexit == nil {
		fexit = os.Exit
	}
	return fexit
}

func safePanic(fpanic func(string)) func(string) {
	if fpanic == nil {
		fpanic = func(m string) { panic(m) }
	}
	return fpanic
}

func exitLogger(logs logger.Logger, fexit func(int)) logger.Logger {
	return logger.LoggerFunc(func(m string, a ...interface{}) {
		defer safeExit(fexit)(ExitCode)
		logs.Logf(m, a...)
	})
}

func panicLogger(logs logger.Logger, fpanic func(string)) logger.Logger {
	return logger.LoggerFunc(func(m string, a ...interface{}) {
		defer safePanic(fpanic)(m)
		logs.Logf(m, a...)
	})
}

type StreamOrLogger struct {
	io.Stream
	logger.Logger
}

type Config struct {
	Level levels.Level
	Sink  StreamOrLogger

	// Exit, when unset, will invoke os.Exit upon calls to Fatalf
	Exit func(int)

	// Panic, when unset, will invoke golang's panic(string) upon calls to Panicf
	Panic func(string)

	// Decorators are applied to the underlying Sink.Stream (never to Sink.Logger)
	Decorators io.Decorators

	// Marshals a log event to an underlying Sink.Stream, defaults to io.Printf.
	// All marshalers should invoke Stream.EOM after processing each log event.
	Marshaler io.StreamOp
}

// NoPanic generates a noop panic func
func NoPanic() func(string) { return func(string) {} }

// NoExit generates a noop exit func
func NoExit() func(int) { return func(int) {} }

var (
	_ = &Config{Panic: NoPanic()} // NoPanic is a panic func generator
	_ = &Config{Exit: NoExit()}   // NoExit is an exit func generator

	DefaultConfig = Config{
		Level: levels.Info, // Level defaults to levels.Info
	}

	// Default logs everything "info" and higher ("warn", "error", ...) to SystemLogger
	Default = func() (i levels.Interface) { i, _ = DefaultConfig.With(NoOption()); return }()
)

// Option is a functional option interface for making changes to a Config
type Option func(*Config) Option

// NoOption returns an option that doesn't make any changes to a Config.
// Use to improve readability or as a noop default Option.
func NoOption() (opt Option) {
	opt = Option(func(_ *Config) Option { return opt })
	return
}

func (cfg Config) With(opt ...Option) (levels.Interface, Option) {
	return cfg.WithContext(io.NoContext(), opt...)
}

func (cfg Config) WithContext(ctx io.Context, opt ...Option) (levels.Interface, Option) {
	lastOpt := NoOption()
	for _, o := range opt {
		if o != nil {
			lastOpt = o(&cfg)
		}
	}
	t := levels.Transform{
		levels.Fatal: func(x logger.Logger) logger.Logger { return exitLogger(x, cfg.Exit) },
		levels.Panic: func(x logger.Logger) logger.Logger { return panicLogger(x, cfg.Panic) },
	}
	if cfg.Sink.Stream != nil {
		return LeveledStreamer(ctx, cfg.Level, cfg.Sink.Stream, cfg.Marshaler, t, cfg.Decorators...), lastOpt
	}
	return LeveledLogger(cfg.Level, cfg.Sink.Logger, t), lastOpt
}

func Level(level levels.Level) Option {
	return func(c *Config) Option {
		old := c.Level
		c.Level = level
		return Level(old)
	}
}

func Sink(x StreamOrLogger) Option {
	return func(c *Config) Option {
		old := c.Sink
		c.Sink = x
		return Sink(old)
	}
}

func Stream(stream io.Stream) Option {
	return Sink(StreamOrLogger{Stream: stream})
}

func Logger(logs logger.Logger) Option {
	return Sink(StreamOrLogger{Logger: logs})
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

func Marshaler(m io.StreamOp) Option {
	return func(c *Config) Option {
		old := c.Marshaler
		c.Marshaler = m
		return Marshaler(old)
	}
}

// Decorate returns a functional Option that appends the given decorators to the Config.
func Decorate(d ...io.Decorator) Option {
	return func(c *Config) Option {
		var old io.Decorators
		if n := len(c.Decorators); n > 0 {
			old = make(io.Decorators, n)
			copy(old, c.Decorators)
		}
		c.Decorators = append(c.Decorators, d...)

		// the undo option should copy back the old
		// decorators exactly as they were
		return Option(func(c *Config) Option {
			c.Decorators = old
			return Decorate(d...)
		})
	}
}
