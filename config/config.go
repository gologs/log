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
	"fmt"
	"os"

	"github.com/jdef/log/io"
	"github.com/jdef/log/logger"
)

// TODO(jdef) need to make this thread safe

// ExitCode is passed to exit functions that are invoked upon calls to Fatalf
var ExitCode = 1

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
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

func (min Level) Filter(at Level) io.Decorator {
	return func(op io.StreamOp) io.StreamOp {
		if at >= min {
			return op
		}
		return io.NullOp()
	}
}

func (min Level) Logger(logs logger.Logger, at Level) logger.Logger {
	if at >= min {
		return logs
	}
	return logger.Null()
}

var levelCodes = map[Level][]byte{
	LevelDebug: []byte("D"),
	LevelInfo:  []byte("I"),
	LevelWarn:  []byte("W"),
	LevelError: []byte("E"),
	LevelFatal: []byte("F"),
	LevelPanic: []byte("P"),
}

// TODO(jdef) test this
func (x Level) Annotated() io.Decorator {
	code, ok := levelCodes[x]
	if !ok {
		// fail fast
		panic(fmt.Sprintf("unexpected level: %q", x))
	}
	return func(op io.StreamOp) io.StreamOp {
		return func(c io.Context, s io.Stream, m string, a ...interface{}) (err error) {
			if _, err = s.Write(code); err == nil {
				err = op(c, s, m, a...)
			}
			return
		}
	}
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

func WithLevelLoggers(debugf, infof, warnf, errorf, fatalf, panicf logger.Logger) Interface {
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

func LeveledStreamer(
	ctx io.Context,
	min Level,
	s io.Stream,
	fexit func(int),
	fpanic func(string),
	d ...io.Decorator,
) Interface {
	op := io.Operator(ctx)
	if s == nil {
		s = io.SystemStream()
	}
	exitDecorator := io.Decorator(func(op io.StreamOp) io.StreamOp {
		return func(c io.Context, w io.Stream, m string, a ...interface{}) (err error) {
			defer safeExit(fexit)(ExitCode)
			return op(c, w, m, a...)
		}
	})
	panicDecorator := io.Decorator(func(op io.StreamOp) io.StreamOp {
		return func(c io.Context, w io.Stream, m string, a ...interface{}) (err error) {
			defer safePanic(fpanic)(m)
			return op(c, w, m, a...)
		}
	})

	applyAnnotations := false
	if len(d) == 0 {
		applyAnnotations = true
	}

	logAt := func(level Level, d ...io.Decorator) logger.Logger {
		var annotator io.Decorator
		if applyAnnotations {
			annotator = level.Annotated()
		}
		d = append(d, annotator, min.Filter(level))
		return logger.StreamLogger(ctx, s, logger.IgnoreErrors(), op, d...)
	}
	return WithLevelLoggers(
		logAt(LevelDebug, d...),
		logAt(LevelInfo, d...),
		logAt(LevelWarn, d...),
		logAt(LevelError, d...),
		logAt(LevelFatal, append(d, exitDecorator)...),
		logAt(LevelPanic, append(d, panicDecorator)...),
	)
}

func LeveledLogger(min Level, logs logger.Logger, fexit func(int), fpanic func(string)) Interface {
	if logs == nil {
		logs = logger.SystemLogger()
	}
	exitLogger := logger.LoggerFunc(func(m string, a ...interface{}) {
		defer safeExit(fexit)(ExitCode)
		logs.Logf(m, a...)
	})
	panicLogger := logger.LoggerFunc(func(m string, a ...interface{}) {
		defer safePanic(fpanic)(m)
		logs.Logf(m, a...)
	})
	return WithLevelLoggers(
		min.Logger(logs, LevelDebug),
		min.Logger(logs, LevelInfo),
		min.Logger(logs, LevelWarn),
		min.Logger(logs, LevelError),
		min.Logger(exitLogger, LevelFatal),
		min.Logger(panicLogger, LevelPanic),
	)
}

type StreamOrLogger struct {
	io.Stream
	logger.Logger
}

type Config struct {
	Level Level
	Sink  StreamOrLogger

	// Exit, when unset, will invoke os.Exit upon calls to Fatalf
	Exit func(int)

	// Panic, when unset, will invoke golang's panic(string) upon calls to Panicf
	Panic func(string)

	// Decorators are applied to the underlying Sink.Stream (never to Sink.Logger)
	Decorators io.Decorators
}

// NoPanic generates a noop panic func
func NoPanic() func(string) { return func(string) {} }

// NoExit generates a noop exit func
func NoExit() func(int) { return func(int) {} }

var (
	_ = &Config{Panic: NoPanic()} // NoPanic is a panic func generator
	_ = &Config{Exit: NoExit()}   // NoExit is an exit func generator

	DefaultConfig = Config{
		Level: LevelInfo, // Level defaults to LevelInfo
	}

	// Default logs everything "info" and higher ("warn", "error", ...) to SystemLogger
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
	return cfg.WithContext(io.NoContext(), opt...)
}

func (cfg Config) WithContext(ctx io.Context, opt ...Option) (Interface, Option) {
	lastOpt := NoOption()
	for _, o := range opt {
		if o != nil {
			lastOpt = o(&cfg)
		}
	}
	if cfg.Sink.Stream != nil {
		return LeveledStreamer(ctx, cfg.Level, cfg.Sink.Stream, cfg.Exit, cfg.Panic), lastOpt
	}
	return LeveledLogger(cfg.Level, cfg.Sink.Logger, cfg.Exit, cfg.Panic), lastOpt
}

func (level Level) Option() Option {
	return func(c *Config) Option {
		old := c.Level
		c.Level = level
		return old.Option()
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
