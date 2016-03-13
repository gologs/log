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
	"encoding/binary"
	stdio "io"
	"testing"

	. "github.com/gologs/log/io"
)

func TestRecordIO(t *testing.T) {
	const message = "foo"
	var (
		b         bytes.Buffer
		rio       = RecordIO(&b)
		marshaler = Format()
		err       = marshaler(nil, rio, message)
	)
	if err != nil {
		t.Fatal(err)
	}
	n, err := binary.ReadUvarint(&b)
	if err != nil {
		t.Fatal(err)
	}
	if int(n) != len(message) {
		t.Fatalf("expected length %d instead of %d", len(message), n)
	}
	actual := make([]byte, len(message))
	r, err := b.Read(actual)
	if err != nil {
		t.Fatal(err)
	}
	if r != len(message) {
		t.Fatalf("expected length %d instead of %d", len(message), r)
	}
	if string(actual) != message {
		t.Fatalf("expected message %q instead of %q", message, actual)
	}
}

type flakeyWriter int

// Write should return an error if not all bytes are written out; we purposely
// violate that rule here to invoke additional error checking in RecordIO
func (lw *flakeyWriter) Write(b []byte) (x int, err error) {
	if *lw > 0 {
		x = len(b)
		if x < int(*lw) {
			*lw -= flakeyWriter(x)
		} else {
			x = int(*lw)
			*lw = 0
			// would normally set err here
		}
	} // else, would normally set err here
	return
}

func TestRecordIO_WithFlakeyWriter(t *testing.T) {
	const message = "foo"
	var (
		lw        = flakeyWriter(2)
		rio       = RecordIO(&lw)
		marshaler = Format()
		err       = marshaler(nil, rio, message)
	)
	if err != stdio.ErrShortWrite {
		t.Fatalf("unexpected error: %v", err)
	}
}
