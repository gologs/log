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
	"fmt"

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

// TODO(jdef) test this
func (x Level) Annotated() io.Decorator {
	code, ok := levelCodes[x]
	if !ok {
		// fail fast
		panic(fmt.Sprintf("unexpected level: %q", x))
	}
	return func(op io.StreamOp) io.StreamOp {
		return func(c context.Context, s io.Stream, m string, a ...interface{}) (err error) {
			if _, err = s.Write(code); err == nil {
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
