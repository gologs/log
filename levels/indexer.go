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
	"github.com/gologs/log/logger"
)

// Indexer functions map a Level to a Logger, or else return false
type Indexer interface {
	Logger(Level) (logger.Logger, bool)
}

// IndexerFunc is the functional adaptation of the Indexer interface
type IndexerFunc func(Level) (logger.Logger, bool)

// Logger implements Indexer
func (f IndexerFunc) Logger(lvl Level) (logger.Logger, bool) { return f(lvl) }

type levelMap map[Level]logger.Logger

func (lm levelMap) Logger(lvl Level) (logs logger.Logger, ok bool) {
	logs, ok = lm[lvl]
	return
}

// NewIndexer builds a logger for each Level, starting with the original Logger
// in the given Indexer and then applying the provided transforms. If nil is given
// for `levels` then all log levels are assumed.
func NewIndexer(idx Indexer, levels []Level, chain ...TransformOp) Indexer {
	if levels == nil {
		levels = allLevels
	}
	m := make(levelMap, len(levels))
	for _, x := range levels {
		logs, ok := idx.Logger(x)
		if !ok {
			continue
		}
		x, logs = TransformOps(chain).Apply(x, logs)
		m[x] = logs
	}
	return m
}
