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
	"time"

	"github.com/gologs/log/caller"
	"github.com/gologs/log/context"
	"github.com/gologs/log/context/timestamp"
	"github.com/gologs/log/encoding"
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

// LeveledStreamer generates a leveled logging interface for the given io.Stream oriented configuration.
func LeveledStreamer(
	ctx context.Getter,
	min levels.Level,
	s io.Stream,
	marshaler encoding.Marshaler,
	t levels.TransformOps,
	callTracking caller.Tracking,
	errorSink chan<- error,
	builder logger.Builder,
) levels.Interface {
	return leveledLogger(
		safeContext(ctx),
		min,
		safeBuilder(builder)(s, marshaler, errorSink),
		t,
		callTracking)
}

func safeBuilder(b logger.Builder) logger.Builder {
	if b == nil {
		b = logger.WithStream
	}
	return logger.Builder(func(s io.Stream, marshaler encoding.Marshaler, errorSink chan<- error) logger.Logger {
		if s == nil {
			s = io.SystemStream(2) // TODO(jdef) this value is probably garbage
		}
		if errorSink == nil {
			errorSink = logger.IgnoreErrors()
		}
		return b(s, marshaler, errorSink)
	})
}

// LeveledLogger generates a leveled logging interface for the given logger.Logger oriented configuration.
func LeveledLogger(
	ctx context.Getter,
	min levels.Level,
	logs logger.Logger,
	t levels.TransformOps,
	callTracking caller.Tracking,
) levels.Interface {
	if logs == nil {
		logs = logger.SystemLogger()
	}
	return leveledLogger(safeContext(ctx), min, logs, t, callTracking)
}

func leveledLogger(
	ctx context.Getter,
	min levels.Level,
	logs logger.Logger,
	t levels.TransformOps,
	callTracking caller.Tracking,
) levels.Interface {
	var (
		logAt = levels.IndexerFunc(func(level levels.Level) (logger.Logger, bool) {
			return logger.WithContext(levels.DecorateContext(level), logs), true
		})
		g lockGuard
	)
	t = append(t, g.Apply)
	if callTracking.Enabled {
		t = append(t,
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
	t = append(t, levels.MinTransform(min))
	ctx = context.NewGetter(ctx, timestamp.NewDecorator(time.Now))
	return levels.WithLoggers(ctx, levels.NewIndexer(logAt, nil, t...))
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

	// Decorators are applied to the Stream (never to Sink.Logger)
	Decorators encoding.Decorators

	// Marshals a log event to Stream, defaults to io.Printf.
	// A Marshaler invokes Stream.EOM as the final step of processing each log event.
	Marshaler encoding.Marshaler

	// Errors receives errors as they occur upon processing streaming events
	// (only applies when using Stream, not for Logger).
	// Defaults to logger.IgnoreErrors().
	Errors chan<- error

	// Builder generates a Logger using the configured Stream, Marshaler, and Errors
	Builder logger.Builder
}

// Config is a complete logging configuration. Fields may be tweaked manually, or by way
// of functional Option funcs.
type Config struct {
	// Context returns the func that generates a context for each log event. This func
	// is invoked once for every log event and must be safe to execute concurrently.
	Context context.Getter

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

	// TransformOps allow clients to highly customize log processing based on levels
	TransformOps levels.TransformOps
}

// NoPanic generates a noop panic func
func NoPanic() func(string) { return func(string) {} }

// NoExit generates a noop exit func
func NoExit() func(int) { return func(int) {} }

var (
	_ = &Config{Panic: NoPanic()} // NoPanic is a panic func generator
	_ = &Config{Exit: NoExit()}   // NoExit is an exit func generator

	// DefaultConfig is used to generate the initial value for Current.
	DefaultConfig = Porcelain()

	// Logging is a logging instance constructed with default configuration:
	// it logs everything "info" and higher ("warn", "error", ...) to logger.SystemLogger()
	Logging = func() (i levels.Interface) { i, _ = DefaultConfig.With(NoOption()); return }()
)

// Porcelain returns a cleanroom, configuration.
func Porcelain() Config {
	return Config{
		Level:    levels.Info, // Level defaults to levels.Info
		ExitCode: 1,           // ExitCode defaults to 1
		CallTracking: caller.Tracking{
			Enabled: true,
			Depth:   DefaultCallerDepth,
		},
	}
}

// Option is a functional option interface for making changes to a Config
type Option func(*Config) Option

// NoOption returns an option that doesn't make any changes to a Config.
// Use to improve readability or as a noop default Option.
func NoOption() (opt Option) {
	opt = Option(func(_ *Config) Option { return opt })
	return
}

func safeMarshaler(m encoding.Marshaler) encoding.Marshaler {
	if m == nil {
		return encoding.Format()
	}
	return m
}

func safeContext(f context.Getter) context.Getter {
	if f == nil {
		return context.TODO
	}
	return f
}

// With generates a logging interface using the receiving configuration with the given Options applied.
func (cfg Config) With(opt ...Option) (levels.Interface, Option) {
	rollback := Set(cfg)
	for _, o := range opt {
		if o != nil {
			_ = o(&cfg)
		}
	}
	// exit and panic wrappers are always applied after user ops
	t := append(cfg.TransformOps, (&levels.Transform{
		levels.Fatal: func(x logger.Logger) logger.Logger {
			return exitLogger(x, cfg.Exit, cfg.ExitCode)
		},
		levels.Panic: func(x logger.Logger) logger.Logger {
			return panicLogger(x, cfg.Panic)
		},
	}).Apply)
	if cfg.Sink.Stream != nil {
		return LeveledStreamer(
			cfg.Context,
			cfg.Level,
			cfg.Sink.Stream,
			cfg.Sink.Decorators.Decorate(safeMarshaler(cfg.Sink.Marshaler)),
			t,
			cfg.CallTracking,
			cfg.Sink.Errors,
			cfg.Sink.Builder), rollback
	}
	return LeveledLogger(
		cfg.Context,
		cfg.Level,
		cfg.Sink.Logger,
		t,
		cfg.CallTracking), rollback
}

// Copy returns a deep copy of the current config
func (cfg Config) Copy() Config {
	clone := cfg
	clone.Sink.Decorators = cfg.Sink.Decorators.Copy()
	return clone
}

// Set returns a functional Option that sets the entire configuration to that specified.
func Set(cfg Config) Option {
	return func(c *Config) Option {
		old := c.Copy()
		*c = cfg.Copy()
		return Set(old)
	}
}

// Context returns a functional Option that sets the Context generator func.
func Context(f context.Getter) Option {
	return func(c *Config) Option {
		old := c.Context
		c.Context = f
		return Context(old)
	}
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
// destination for log messages. Note: if the sink has a non-nil value setting this Option
// will override it.
func Stream(stream io.Stream) Option {
	return func(c *Config) Option {
		old := c.Sink.Stream
		c.Sink.Stream = stream
		return Stream(old)
	}
}

// Logger is a functional configuration Option that establishes the given logger.Logger as the
// destination for log messages. Note: changing the logger has no effect if the sink's Stream
// field is non-nil.
func Logger(logs logger.Logger) Option {
	return func(c *Config) Option {
		old := c.Sink.Logger
		c.Sink.Logger = logs
		return Logger(old)
	}
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
func Marshaler(m encoding.Marshaler) Option {
	return func(c *Config) Option {
		old := c.Sink.Marshaler
		c.Sink.Marshaler = m
		return Marshaler(old)
	}
}

// Encoding returns a functional Option that appends the given encoding `Decorator`s to what's
// currently configured.
func Encoding(d ...encoding.Decorator) Option {
	return func(c *Config) Option {
		old := c.Sink.Decorators.Copy()
		c.Sink.Decorators = append(c.Sink.Decorators, d...)

		// the undo option should copy back the old
		// decorators exactly as they were
		return Option(func(c *Config) Option {
			c.Sink.Decorators = old
			return Encoding(d...)
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

// Errors returns a functional Option that establishes a consumer of errors generated by the
// logging subsystem.
func Errors(es chan<- error) Option {
	return func(c *Config) Option {
		old := c.Sink.Errors
		c.Sink.Errors = es
		return Errors(old)
	}
}

// Builder returns a functional Option that generates a logger Builder using the Stream-related
// config settings.
func Builder(b logger.Builder) Option {
	return func(c *Config) Option {
		old := c.Sink.Builder
		c.Sink.Builder = b
		return Builder(old)
	}
}

// TransformOps returns a functional Option that appends the given transform operators to those
// already defined for the config.
func TransformOps(ops ...levels.TransformOp) Option {
	return func(c *Config) Option {
		old := c.TransformOps.Copy()
		c.TransformOps = append(c.TransformOps, ops...)

		// undo Option should copy back the old ops exactly as they were before
		return Option(func(c *Config) Option {
			c.TransformOps = old
			return TransformOps(ops...)
		})
	}
}

// AddContext returns a functional Option that applies the given context decorators to the context
// generated by the current getter.
func AddContext(d ...context.Decorator) Option {
	return func(c *Config) Option {
		old := c.Context
		c.Context = context.NewGetter(c.Context, d...)
		return Context(old)
	}
}
