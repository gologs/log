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

package logger_test

import (
	"errors"
	"testing"

	"github.com/gologs/log/context"
	"github.com/gologs/log/io"
	. "github.com/gologs/log/logger"
)

func TestNull(_ *testing.T) {
	// should execute without error
	n := Null()
	n.Logf(nil, "")
}

func TestMulti(t *testing.T) {
	var (
		logToString = func(dest *string) Logger {
			if dest == nil {
				return Null()
			}
			return Func(func(_ context.Context, m string, _ ...interface{}) { *dest = m })
		}
		outputA, outputB string
		a                = logToString(&outputA)
		b                = logToString(&outputB)
		tee              = Multi(a, b)
	)
	tee.Logf(nil, "foo")
	if outputA != "foo" {
		t.Errorf("expected foo for outputA instead of %q", outputA)
	}
	if outputB != "foo" {
		t.Errorf("expected foo for outputB instead of %q", outputB)
	}
}

func TestWithStream(t *testing.T) {
	var (
		marshaler = io.Format()
		output    string
		buf       = &io.BufferedStream{EOMFunc: func(b io.Buffer, _ error) { output = b.String() }}
		logs      = WithStream(buf, marshaler, IgnoreErrors())
	)
	logs.Logf(nil, "foo")
	if output != "foo" {
		t.Errorf("expected foo instead of %q", output)
	}
}

func TestContext(t *testing.T) {
	var (
		d = context.Decorator(func(c context.Context) context.Context {
			return context.WithValue(c, "foo", "bar")
		})
		foo        string
		captureFoo = Func(func(c context.Context, _ string, _ ...interface{}) {
			f, ok := c.Value("foo").(string)
			if ok {
				foo = f
			}
		})
		logs = Context(d)(captureFoo)
	)
	logs.Logf(context.TODO(), "")
	if foo != "bar" {
		t.Errorf("expected bar instead of %q", foo)
	}
}

func TestWithStream_WithError(t *testing.T) {
	var (
		expectedErr = errors.New("some error")
		eomError    error
		marshaler   = io.StreamOp(
			func(_ context.Context, w io.Stream, _ string, _ ...interface{}) error {
				w.EOM(expectedErr)
				return expectedErr
			})
		output string
		buf    = &io.BufferedStream{EOMFunc: func(b io.Buffer, err error) {
			output = b.String()
			eomError = err
		}}
		errCh = make(chan error, 1)
		logs  = WithStream(buf, marshaler, errCh)
	)
	logs.Logf(context.TODO(), "foo") // can't use plain "nil" context if you want error handling
	if output != "" {
		t.Errorf("expected empty output instead of %q", output)
	}
	if eomError != expectedErr {
		t.Errorf("expected %v error instead of %v", expectedErr, eomError)
	}
	select {
	case err := <-errCh:
		if err != expectedErr {
			t.Errorf("expected %v error instead of %v", expectedErr, err)
		}
	default:
		t.Errorf("expected error but got none")
	}
}
