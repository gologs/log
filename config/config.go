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
	"sync"

	"github.com/jdef/log/context"
	"github.com/jdef/log/io"
	"github.com/jdef/log/levels"
	"github.com/jdef/log/logger"
)

type lockGuard struct{ sync.Mutex }

func (g *lockGuard) Apply(x levels.Level, logs logger.Logger) (levels.Level, logger.Logger) {
	return x, logger.LoggerFunc(func(c context.Context, m string, a ...interface{}) {
		g.Lock()
		defer g.Unlock()
		logs.Logf(c, m, a...)
	})
}

// Apply is a levels.TransformOp
var _ = levels.TransformOp((&lockGuard{}).Apply)

func addLevelToContext(x levels.Level) logger.Decorator {
	return logger.Context(func(c context.Context) context.Context {
		return x.NewContext(c)
	})
}

// GenerateLevelLoggers builds a logger for every known log level; for each level
// create a seed logger and apply chain funcs. The results may be fed directly into
// levels.WithLoggers.
func GenerateLevelLoggers(
	ctx context.Context,
	seed func(levels.Level) logger.Logger,
	chain ...levels.TransformOp,
) (_ context.Context, _, _, _, _, _, _ logger.Logger) {

	m := map[levels.Level]logger.Logger{}

	for _, x := range levels.Levels() {
		logs := seed(x)
		for _, c := range chain {
			x, logs = c(x, logs)
		}
		m[x] = logs
	}
	return ctx,
		m[levels.Debug],
		m[levels.Info],
		m[levels.Warn],
		m[levels.Error],
		m[levels.Fatal],
		m[levels.Panic]
}

func LeveledStreamer(
	ctx context.Context,
	min levels.Level,
	s io.Stream,
	marshaler io.StreamOp,
	t levels.Transform,
	decorators ...io.Decorator,
) levels.Interface {
	if ctx == nil {
		ctx = context.None()
	}
	if marshaler == nil {
		marshaler = io.Printf(ctx)
	}
	if s == nil {
		s = io.SystemStream()
	}
	if len(decorators) == 0 {
		decorators = io.Decorators{levels.Annotator()}
	}

	logs := logger.WithStream(
		s,
		logger.IgnoreErrors(),
		io.Decorators(decorators).Decorate(marshaler),
	)
	return leveledLogger(ctx, min, logs, t)
}

func LeveledLogger(ctx context.Context, min levels.Level, logs logger.Logger, t levels.Transform) levels.Interface {
	if ctx == nil {
		ctx = context.None()
	}
	if logs == nil {
		logs = logger.SystemLogger()
	}
	return leveledLogger(ctx, min, logs, t)
}

func leveledLogger(ctx context.Context, min levels.Level, logs logger.Logger, t levels.Transform) levels.Interface {
	var (
		logAt = func(level levels.Level) logger.Logger {
			return addLevelToContext(level)(logs)
		}
		g lockGuard
	)
	return levels.WithLoggers(GenerateLevelLoggers(ctx, logAt, t.Apply, g.Apply, min.Min()))
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

func exitLogger(logs logger.Logger, fexit func(int), code int) logger.Logger {
	return logger.LoggerFunc(func(c context.Context, m string, a ...interface{}) {
		defer safeExit(fexit)(code)
		logs.Logf(c, m, a...)
	})
}

func panicLogger(logs logger.Logger, fpanic func(string)) logger.Logger {
	return logger.LoggerFunc(func(c context.Context, m string, a ...interface{}) {
		defer safePanic(fpanic)(m)
		logs.Logf(c, m, a...)
	})
}

type StreamOrLogger struct {
	io.Stream
	logger.Logger
}

type Config struct {
	Level levels.Level
	Sink  StreamOrLogger

	// ExitCode is passed to exit functions that are invoked upon calls to Fatalf
	ExitCode int

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
		Level:    levels.Info, // Level defaults to levels.Info
		ExitCode: 1,           // ExitCode defaults to 1
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
	return cfg.WithContext(context.None(), opt...)
}

func (cfg Config) WithContext(ctx context.Context, opt ...Option) (levels.Interface, Option) {
	lastOpt := NoOption()
	for _, o := range opt {
		if o != nil {
			lastOpt = o(&cfg)
		}
	}
	t := levels.Transform{
		levels.Fatal: func(x logger.Logger) logger.Logger { return exitLogger(x, cfg.Exit, cfg.ExitCode) },
		levels.Panic: func(x logger.Logger) logger.Logger { return panicLogger(x, cfg.Panic) },
	}
	if cfg.Sink.Stream != nil {
		return LeveledStreamer(ctx, cfg.Level, cfg.Sink.Stream, cfg.Marshaler, t, cfg.Decorators...), lastOpt
	}
	return LeveledLogger(ctx, cfg.Level, cfg.Sink.Logger, t), lastOpt
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

func ExitCode(code int) Option {
	return func(c *Config) Option {
		old := c.ExitCode
		c.ExitCode = code
		return ExitCode(old)
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
