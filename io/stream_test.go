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
	"errors"
	"testing"

	. "github.com/gologs/log/io"
)

func TestNull(t *testing.T) {
	n := Null()
	err := n.EOM(nil)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	x, err := n.Write(nil)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if x != 0 {
		t.Fatalf("unexpected bytes written: %d", x)
	}

	x, err = n.Write(make([]byte, 13))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if x != 13 {
		t.Fatalf("unexpected bytes written: %d", x)
	}
}

func TestBufferedStream(t *testing.T) {
	// other tests already use BufferedStream w/ non-nil EOMFunc's; let's test
	// without an EOMFunc here
	var (
		b           BufferedStream
		expectedErr = errors.New("foo")
		err         = b.EOM(nil)
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	err = b.EOM(expectedErr)
	if err != expectedErr {
		t.Fatalf("unexpected err %v", err)
	}
}
