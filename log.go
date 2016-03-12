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

func Debugf(msg string, args ...interface{}) { config.Default.Debugf(msg, args...) }
func Infof(msg string, args ...interface{})  { config.Default.Infof(msg, args...) }
func Warnf(msg string, args ...interface{})  { config.Default.Warnf(msg, args...) }
func Errorf(msg string, args ...interface{}) { config.Default.Errorf(msg, args...) }
func Fatalf(msg string, args ...interface{}) { config.Default.Fatalf(msg, args...) }
func Panicf(msg string, args ...interface{}) { config.Default.Panicf(msg, args...) }

// Logf is an alias for Infof
func Logf(msg string, args ...interface{}) { config.Default.Infof(msg, args...) }
