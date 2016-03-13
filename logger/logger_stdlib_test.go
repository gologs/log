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
	"bytes"
	"flag"
	"log"
	"os"
	"testing"

	. "github.com/gologs/log/logger"
)

func TestSystemLogger(t *testing.T) {
	const message = "hello stdlib log"
	var (
		syslog   = SystemLogger()
		buf      bytes.Buffer
		expected = message + "\n"
	)
	log.SetOutput(&buf)
	syslog.Logf(nil, message)
	actual := buf.String()
	if actual != expected {
		t.Fatalf("expected %q instead of %q", expected, actual)
	}

	buf.Reset()
	syslog.Logf(nil, "abc %d%d%d", 1, 2, 3)
	expected = "abc 123\n"
	actual = buf.String()
	if actual != expected {
		t.Fatalf("expected %q instead of %q", expected, actual)
	}

	buf.Reset()
	syslog.Logf(nil, "", 1, 2, 3)
	expected = "1 2 3\n"
	actual = buf.String()
	if actual != expected {
		t.Fatalf("expected %q instead of %q", expected, actual)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	log.SetFlags(0)
	os.Exit(m.Run())
}
