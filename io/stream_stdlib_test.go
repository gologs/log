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

package io_test

import (
	"bytes"
	"errors"
	"flag"
	"log"
	"os"
	"testing"

	"github.com/gologs/log/context"
	. "github.com/gologs/log/io"
)

func TestSystemStream(t *testing.T) {
	const message = "hello stdlib log"
	var (
		syslog    = SystemStream(0)
		buf       bytes.Buffer
		expected  = message + "\n"
		marshaler = Format()
	)
	log.SetOutput(&buf)
	err := marshaler(nil, syslog, message)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	actual := buf.String()
	if actual != expected {
		t.Fatalf("expected %q instead of %q", expected, actual)
	}

	buf.Reset()
	err = marshaler(nil, syslog, "abc %d%d%d", 1, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected = "abc 123\n"
	actual = buf.String()
	if actual != expected {
		t.Fatalf("expected %q instead of %q", expected, actual)
	}

	buf.Reset()
	err = marshaler(nil, syslog, "", 1, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected = "1 2 3\n"
	actual = buf.String()
	if actual != expected {
		t.Fatalf("expected %q instead of %q", expected, actual)
	}

	expectedErr := errors.New("someExpectedError")
	marshaler = StreamOp(func(c context.Context, st Stream, m string, a ...interface{}) error {
		// ignore the message, just generate an error
		return st.EOM(expectedErr)
	})
	err = marshaler(nil, syslog, "")
	if err != expectedErr {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	log.SetFlags(0)
	os.Exit(m.Run())
}
