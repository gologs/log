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

package log

import (
	"github.com/gologs/log/config"
)

// this is rubbish, but it silences "go vet"s complaints about lack of format specifiers,
// and it's a dumb enough func that the golang toolchain can optimize this away
func govetIgnoreFormat() string { return "" }

// Debugf logs at levels.Debug
func Debugf(msg string, args ...interface{}) { config.Default.Debugf(msg, args...) }

// Debug logs at levels.Debug
func Debug(args ...interface{}) { config.Default.Debugf("", args...) }

// Infof logs at levels.Info
func Infof(msg string, args ...interface{}) { config.Default.Infof(msg, args...) }

// Info logs at levels.Info
func Info(args ...interface{}) { config.Default.Infof("", args...) }

// Warnf logs at levels.Warn
func Warnf(msg string, args ...interface{}) { config.Default.Warnf(msg, args...) }

// Warn logs at levels.Warn
func Warn(args ...interface{}) { config.Default.Warnf("", args...) }

// Errorf logs at levels.Error
func Errorf(msg string, args ...interface{}) { config.Default.Errorf(msg, args...) }

// Error logs at levels.Error
func Error(args ...interface{}) { config.Default.Errorf(govetIgnoreFormat(), args...) }

// Fatalf logs at levels.Fatal
func Fatalf(msg string, args ...interface{}) { config.Default.Fatalf(msg, args...) }

// Fatal logs at levels.Fatal
func Fatal(args ...interface{}) { config.Default.Fatalf(govetIgnoreFormat(), args...) }

// Panicf logs at levels.Panic
func Panicf(msg string, args ...interface{}) { config.Default.Panicf(msg, args...) }

// Panic logs at levels.Panic
func Panic(args ...interface{}) { config.Default.Panicf(govetIgnoreFormat(), args...) }

// Logf is an alias for Infof
func Logf(msg string, args ...interface{}) { config.Default.Infof(msg, args...) }

// Log is an alias for Info
func Log(args ...interface{}) { config.Default.Infof("", args...) }
