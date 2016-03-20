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

// Transform collects Decorators that are applied to `Logger`s for specific `Level`s.
type Transform map[Level]logger.Decorator

// Apply decorates the given Logger using the Decorator as specified for the given
// Level (via the receiving Transform)
func (t Transform) Apply(x Level, logs logger.Logger) (Level, logger.Logger) {
	if f, ok := t[x]; ok {
		return x, f(logs)
	}
	return x, logs
}

// TransformOp typically returns the same Level with a modified Logger
type TransformOp func(Level, logger.Logger) (Level, logger.Logger)

// TransformOps aggregates TransformOp and allows such operators to be applied in bulk
type TransformOps []TransformOp

// Apply executes all the TransformOps against the input, first to last, and returns the result
func (ops TransformOps) Apply(x Level, logs logger.Logger) (Level, logger.Logger) {
	for _, t := range ops {
		if t != nil {
			x, logs = t(x, logs)
		}
	}
	return x, logs
}

// Copy returns a clone of the ops slice that's independent of the original
func (ops TransformOps) Copy() TransformOps {
	if ops == nil {
		return ops
	}
	clone := make(TransformOps, len(ops))
	copy(clone, ops)
	return clone
}
