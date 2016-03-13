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

package encoding_test

import (
	"errors"
	"testing"

	"github.com/gologs/log/context"
	. "github.com/gologs/log/encoding"
	"github.com/gologs/log/io"
)

func TestNullMarshaler(t *testing.T) {
	var (
		n   = NullMarshaler()
		err = n(nil, nil, "")
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = n(nil, nil, "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = n(nil, nil, "", 1, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = n(nil, nil, "abc", 1, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNoDecorator(t *testing.T) {
	var (
		n   = NullMarshaler()
		err = n(nil, nil, "")
	)
	err = n(nil, nil, "abc", 1, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n2 := NoDecorator()(n)
	// cannot test n == n2 because golang won't let me
	err = n2(nil, nil, "abc", 1, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorators(t *testing.T) {
	var (
		n   = NullMarshaler()
		dd  = Decorators{NoDecorator(), nil, NoDecorator()}
		err = dd.Decorate(n)(nil, nil, "abc", 1, 2, 3)
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	foo := "bar"
	dd[2] = Decorator(func(op Marshaler) Marshaler {
		return func(_ context.Context, _ io.Stream, m string, _ ...interface{}) error {
			foo = m
			return nil
		}
	})
	n = dd.Decorate(n)
	err = n(nil, nil, "qaz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if foo != "qaz" {
		t.Fatalf("expected qaz instead of %q", foo)
	}
}

func TestPrefix(t *testing.T) {
	var (
		d           = Prefix(nil)
		n           = NullMarshaler()
		err         = d(n)(nil, nil, "")
		expectedErr = errors.New("someError")
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	d = Prefix(func(_ context.Context) ([]byte, error) {
		return nil, expectedErr
	})
	err = d(n)(nil, nil, "")
	if err != expectedErr {
		t.Fatalf("unexpected error: %v", err)
	}

	d = Prefix(func(_ context.Context) ([]byte, error) {
		return []byte("foo"), expectedErr
	})
	capture := ""
	b := &io.BufferedStream{
		EOMFunc: func(buf io.Buffer, e error) error {
			if e != nil {
				return e
			}
			capture = buf.String()
			return nil
		},
	}
	err = Format(d)(nil, b, "")
	if err != expectedErr {
		t.Fatalf("unexpected error: %v", err)
	}
	if capture != "" {
		t.Fatalf("unexpected capture: %q", capture)
	}

	d = Prefix(func(_ context.Context) ([]byte, error) {
		return []byte("bar"), nil
	})
	err = Format(d)(nil, b, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capture != "barfoo" {
		t.Fatalf("unexpected capture: %q", capture)
	}

	err = Format(d)(nil, b, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capture != "bar" {
		t.Fatalf("unexpected capture: %q", capture)
	}
}

func TestWithContext(t *testing.T) {
	var (
		n   = NullMarshaler()
		d   = WithContext(nil)
		err = d(n)(nil, nil, "")
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cd := context.NewDecorator("foo", "bar")
	d = WithContext(cd)
	foo := ""
	d2 := Decorator(func(op Marshaler) Marshaler {
		return func(c context.Context, _ io.Stream, m string, _ ...interface{}) error {
			foo = c.Value("foo").(string)
			return nil
		}
	})
	err = Decorators{d2, d}.Decorate(n)(context.Background(), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if foo != "bar" {
		t.Fatalf("unexpected foo: %q", foo)
	}
}
