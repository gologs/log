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

package timestamp

import (
	"time"

	"github.com/gologs/log/context"
)

type key int

const (
	tsKey key = iota
)

// Clock functions return the current time
type Clock func() time.Time

// FromContext extracts a timestamp from the provided context.
func FromContext(ctx context.Context) (t time.Time, ok bool) {
	t, ok = ctx.Value(tsKey).(time.Time)
	return
}

// NewContext returns a Context that contains the provided timestamp.
func NewContext(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, tsKey, t)
}
