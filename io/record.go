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

package io

import (
	"bytes"
	"encoding/binary"
	"io"
)

type recordStream struct {
	delegate io.Writer
	buf      bytes.Buffer
	sz       [binary.MaxVarintLen64]byte
}

func (rs *recordStream) Write(b []byte) (int, error) {
	return rs.buf.Write(b)
}

func (rs *recordStream) EOM(err error) error {
	defer rs.buf.Reset()
	if err != nil {
		return err
	}

	// encode and write the length bytes
	buflen := rs.buf.Len()
	n := binary.PutUvarint(rs.sz[:], uint64(buflen))
	w, err := rs.delegate.Write(rs.sz[:n])
	if err != nil {
		return err
	}
	if w != n {
		// should never happen..
		return io.ErrShortWrite
	}

	// write the data bytes
	written, err := rs.buf.WriteTo(rs.delegate)
	if err != nil {
		return err
	}
	if written != int64(buflen) {
		// should never happen
		return io.ErrShortWrite
	}
	return nil
}

// RecordIO returns a stream that writes log messages to the underlying stream, each
// message prefixed with length bytes generated by binary.PutUvarint.
func RecordIO(w io.Writer) Stream { return &recordStream{delegate: w} }
