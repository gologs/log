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

// Filter returns true if the given level is accepted
type Filter func(Level) bool

// Or generates a logical OR of the receiving Filter and the other specified as a param.
// If both the receiver and other are nil then the returned filter accepts all Levels.
func (f Filter) Or(other Filter) Filter {
	if f2, ok := reduceNilFilter(f, other); ok {
		return f2
	}
	return func(lvl Level) bool {
		return f(lvl) || other(lvl)
	}
}

// And generates a logical AND of the receiving Filter and the other specified as a param.
// If both the receiver and other are nil then the returned filter accepts all Levels.
func (f Filter) And(other Filter) Filter {
	if f2, ok := reduceNilFilter(f, other); ok {
		return f2
	}
	return func(lvl Level) bool {
		return f(lvl) && other(lvl)
	}
}

// Xor generates a logical XOR (eXclusive-OR) of the receiving Filter and the other specified
// as a param. If both the receiver and other are nil then the returned filter accepts all Levels.
func (f Filter) Xor(other Filter) Filter {
	if f2, ok := reduceNilFilter(f, other); ok {
		return f2
	}
	return func(lvl Level) bool {
		a, b := f(lvl), other(lvl)
		return (a || b) && !(a && b)
	}
}

func reduceNilFilter(a, b Filter) (Filter, bool) {
	if a == nil {
		if b == nil {
			// both args are nil, accept everything
			return func(_ Level) bool { return true }, true
		}
		return b, true
	}
	if b == nil {
		return a, true
	}
	return nil, false
}

// MatchAny filters return true if the logical AND of a level with the given levelMask is non-zero
func MatchAny(levelMask Level) Filter { return func(x Level) bool { return (levelMask & x) != 0 } }

// MatchExact filters return true if a tested level is identical to the level provided to the matcher.
func MatchExact(lvl Level) Filter { return func(x Level) bool { return x == lvl } }

// MatchAtOrAbove filters return true if the tested level is the same or higher then that provided
// to the matcher.
func MatchAtOrAbove(lvl Level) Filter { return func(x Level) bool { return x >= lvl } }

// Broadcast replicates log messages for the accepted levels to all the provided loggers.
// If replace is false, a copy of the log message is also sent to the original input logger
// of the returned TransformOp. If replace is true and len(log) == 0 then accepted logs
// events will be dropped (sent to logger.Null()). Log messages that are not accepted by
// the filter are simply passed through the original logger.
func Broadcast(filter Filter, replace bool, log ...logger.Logger) TransformOp {
	return func(x Level, ll logger.Logger) (Level, logger.Logger) {
		if filter(x) {
			if replace {
				if len(log) == 0 {
					return x, logger.Null()
				}
				return x, logger.Multi(log...)
			}
			if len(log) == 0 {
				return x, ll // edge case, but there's no use wrapping here
			}
			return x, logger.Multi(append(log, ll)...)
		}
		return x, ll
	}
}

// Accept drops log messages whose log level does not match the given filter.
func Accept(filter Filter) TransformOp {
	return func(x Level, ll logger.Logger) (Level, logger.Logger) {
		if filter(x) {
			return x, ll
		}
		return x, logger.Null()
	}
}
