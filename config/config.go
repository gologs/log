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

	"github.com/gologs/log/caller"
	"github.com/gologs/log/context"
	"github.com/gologs/log/io"
	"github.com/gologs/log/levels"
	"github.com/gologs/log/logger"
)

// DefaultCallerDepth is appropriate when invoking, for example Infof, on the glogs/log
// package directly.
// NOTE: the call-depth specified (5) has been carefully selected; if any transforms are
// introduced that would further wrap the logger that we consume below then the call-depth
// will need to be increased accordingly.
const DefaultCallerDepth = 5

type lockGuard struct{ sync.Mutex }

func (g *lockGuard) Apply(x levels.Level, logs logger.Logger) (levels.Level, logger.Logger) {
	return x, logger.Func(func(c context.Context, m string, a ...interface{}) {
		g.Lock()
		defer g.Unlock()
		logs.Logf(c, m, a...)
	})
}

// Apply is a levels.TransformOp
var _ = levels.TransformOp((&lockGuard{}).Apply)

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

// LeveledStreamer generates a leveled logging interface for the given io.Stream oriented configuration.
func LeveledStreamer(
	ctx context.Context,
	min levels.Level,
	s io.Stream,
	marshaler io.StreamOp,
	t levels.Transform,
	callTracking caller.Tracking,
	errorSink chan<- error,
	decorators ...io.Decorator,
) levels.Interface {
	if ctx == nil {
		ctx = context.Background()
	}
	if marshaler == nil {
		marshaler = io.Format()
	}
	if s == nil {
		s = io.SystemStream(2) // TODO(jdef) this value is probably garbage
	}
	if len(decorators) == 0 {
		decorators = io.Decorators{levels.Annotator()}
	}
	if errorSink == nil {
		errorSink = logger.IgnoreErrors()
	}

	logs := logger.WithStream(
		s,
		io.Decorators(decorators).Decorate(marshaler),
		errorSink,
	)
	return leveledLogger(ctx, min, logs, t, callTracking)
}

// LeveledLogger generates a leveled logging interface for the given logger.Logger oriented configuration.
func LeveledLogger(
	ctx context.Context,
	min levels.Level,
	logs logger.Logger,
	t levels.Transform,
	callTracking caller.Tracking,
) levels.Interface {
	if ctx == nil {
		ctx = context.Background()
	}
	if logs == nil {
		logs = logger.SystemLogger()
	}
	return leveledLogger(ctx, min, logs, t, callTracking)
}

func leveledLogger(
	ctx context.Context,
	min levels.Level,
	logs logger.Logger,
	t levels.Transform,
	callTracking caller.Tracking,
) levels.Interface {
	var (
		logAt = func(level levels.Level) logger.Logger {
			return logger.WithContext(levels.DecorateContext(level), logs)
		}
		g    lockGuard
		tops = []levels.TransformOp{t.Apply, g.Apply}
	)
	if callTracking.Enabled {
		tops = append(tops,
			// inject caller info into context (file/line); this is probably the best place to do it
			// since we can predict the call-depth here and it will work for both Stream- and Logger-
			// based approaches.
			// NOTE: care has been taken to avoid locking the guard Mutex until absolutely necessary.
			// For example, the log level threshold filter and caller injection both execute *before*
			// the mutex is locked (pulling the call stack run the runtime is expensive).
			levels.TransformOp(func(x levels.Level, logs logger.Logger) (levels.Level, logger.Logger) {
				return x, logger.WithContext(caller.WithContext(callTracking), logs)
			}),
		)
	}
	tops = append(tops, levels.MinTransform(min))
	return levels.WithLoggers(GenerateLevelLoggers(ctx, logAt, tops...))
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
	return logger.Func(func(c context.Context, m string, a ...interface{}) {
		defer safeExit(fexit)(code)
		logs.Logf(c, m, a...)
	})
}

func panicLogger(logs logger.Logger, fpanic func(string)) logger.Logger {
	return logger.Func(func(c context.Context, m string, a ...interface{}) {
		defer safePanic(fpanic)(m)
		logs.Logf(c, m, a...)
	})
}

// StreamOrLogger prescribes the destination for log messages. It is expected that clients
// set either Stream or Logger, but not both. If both are set then the factory functions of
// this package prefer the Stream instance.
type StreamOrLogger struct {
	io.Stream
	logger.Logger
}

// Config is a complete logging configuration. Fields may be tweaked manually, or by way
// of functional Option funcs.
type Config struct {
	// Level is the minimum log threshold; messages below this level will be discarded
	Level levels.Level

	// Sink is the destination for log events
	Sink StreamOrLogger

	// CallTracking, when true, queries runtime for the call stack to populate Caller
	// in the logging Context.
	CallTracking caller.Tracking

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

	// ErrorSink receives errors as they occur upon processing streaming events
	// (only applies when using Sink.Stream, not for Sink.Logger).
	// Defaults to logger.IgnoreErrors().
	ErrorSink chan<- error
}

// NoPanic generates a noop panic func
func NoPanic() func(string) { return func(string) {} }

// NoExit generates a noop exit func
func NoExit() func(int) { return func(int) {} }

var (
	_ = &Config{Panic: NoPanic()} // NoPanic is a panic func generator
	_ = &Config{Exit: NoExit()}   // NoExit is an exit func generator

	// DefaultConfig is used to generate the initial Default logger
	DefaultConfig = Config{
		Level:    levels.Info, // Level defaults to levels.Info
		ExitCode: 1,           // ExitCode defaults to 1
		CallTracking: caller.Tracking{
			Enabled: true,
			Depth:   DefaultCallerDepth,
		},
	}

	// Default is a logging instance constructed with default configuration:
	// it logs everything "info" and higher ("warn", "error", ...) to logger.SystemLogger()
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

// With generates a logging interface using the specified functional Options with a
// context.Background()
func (cfg Config) With(opt ...Option) (levels.Interface, Option) {
	return cfg.WithContext(context.Background(), opt...)
}

// WithContext generates a logging interface using the specified Context and functional Options
func (cfg Config) WithContext(ctx context.Context, opt ...Option) (levels.Interface, Option) {
	lastOpt := NoOption()
	for _, o := range opt {
		if o != nil {
			lastOpt = o(&cfg)
		}
	}
	t := levels.Transform{
		levels.Fatal: func(x logger.Logger) logger.Logger {
			return exitLogger(x, cfg.Exit, cfg.ExitCode)
		},
		levels.Panic: func(x logger.Logger) logger.Logger {
			return panicLogger(x, cfg.Panic)
		},
	}
	if cfg.Sink.Stream != nil {
		return LeveledStreamer(
			ctx,
			cfg.Level,
			cfg.Sink.Stream,
			cfg.Marshaler,
			t,
			cfg.CallTracking,
			cfg.ErrorSink,
			cfg.Decorators...), lastOpt
	}
	return LeveledLogger(
		ctx,
		cfg.Level,
		cfg.Sink.Logger,
		t,
		cfg.CallTracking), lastOpt
}

// Level is a functional configuration Option that sets the minimum log level threshold.
func Level(level levels.Level) Option {
	return func(c *Config) Option {
		old := c.Level
		c.Level = level
		return Level(old)
	}
}

// Sink is a functional configuration Option that sets the destination for log messages.
func Sink(x StreamOrLogger) Option {
	return func(c *Config) Option {
		old := c.Sink
		c.Sink = x
		return Sink(old)
	}
}

// Stream is a functional configuration Option that establishes the given io.Stream as the
// destination for log messages.
func Stream(stream io.Stream) Option {
	return Sink(StreamOrLogger{Stream: stream})
}

// Logger is a functional configuration Option that establishes the given logger.Logger as the
// destination for log messages.
func Logger(logs logger.Logger) Option {
	return Sink(StreamOrLogger{Logger: logs})
}

// OnExit is a functional configuration Option that defines the behavior of Exitf after a
// log message has been delivered to the sink.
func OnExit(f func(int)) Option {
	return func(c *Config) Option {
		old := c.Exit
		c.Exit = f
		return OnExit(old)
	}
}

// ExitCode is a functional configuration Option that defines the preferred process exit code
// generated upon process termination via calls to Exitf. Implementations of exit funcs (set via
// OnExit) should report this value.
func ExitCode(code int) Option {
	return func(c *Config) Option {
		old := c.ExitCode
		c.ExitCode = code
		return ExitCode(old)
	}
}

// OnPanic is a functional configuration Option that defines the behavior of Panicf after a
// log message has been delivered to the sink.
func OnPanic(f func(msg string)) Option {
	return func(c *Config) Option {
		old := c.Panic
		c.Panic = f
		return OnPanic(old)
	}
}

// Marshaler is a functional configuration Option that serializes log messages to an io.Stream.
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

// CallTracking returns a functional Option that determines whether logging Context is annotated
// with a caller.Caller, and if so the "caller depth" to use when crawling the runtime call stack.
func CallTracking(t caller.Tracking) Option {
	return func(c *Config) Option {
		old := c.CallTracking
		c.CallTracking = t
		return CallTracking(old)
	}
}

// ErrorSink returns a functional Option that establishes a consumer of errors generated by the
// logging subsystem.
func ErrorSink(es chan<- error) Option {
	return func(c *Config) Option {
		old := c.ErrorSink
		c.ErrorSink = es
		return ErrorSink(old)
	}
}
