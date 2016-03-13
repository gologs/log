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

package ioutil

import (
	"github.com/gologs/log/context"
	"github.com/gologs/log/io"
	"github.com/gologs/log/levels"
)

var levelCodes = map[levels.Level][]byte{
	levels.Debug: []byte("D"),
	levels.Info:  []byte("I"),
	levels.Warn:  []byte("W"),
	levels.Error: []byte("E"),
	levels.Fatal: []byte("F"),
	levels.Panic: []byte("P"),
}

// LevelPrefix generates a stream io.Prefix decorator that prepends a level code
// label to every log message.
func LevelPrefix() io.Decorator {
	return io.Prefix(func(c context.Context) (b []byte, err error) {
		if x, ok := levels.FromContext(c); ok {
			if code, ok := levelCodes[x]; ok {
				b = code
			}
		}
		return
	})
}
