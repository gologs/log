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

package log_test

import (
	"bytes"
	"fmt"

	"github.com/jdef/log"
	"github.com/jdef/log/config"
	"github.com/jdef/log/io"
	"github.com/jdef/log/logger"
)

func Example_withCustomLogger() {
	var (
		logs    = []string{}
		flogger = logger.LoggerFunc(func(m string, a ...interface{}) {
			logs = append(logs, fmt.Sprintf(m, a...))
		})
	)

	// swap out the default logger
	config.Default, _ = config.DefaultConfig.With(config.Logger(flogger))
	log.Debugf("I can count 1 2 %d", 3)
	log.Infof("and more 4 5 %d", 6)

	// print what we logged
	fmt.Printf("%d\n", len(logs))
	for i := range logs {
		fmt.Println(logs[i])
	}

	// Output:
	// 1
	// and more 4 5 6
}

func Example_withCustomStream() {
	var (
		logs   = []string{}
		stream = &io.BufferedStream{
			EOMFunc: func(b *bytes.Buffer, _ error) {
				logs = append(logs, b.String())
			},
		}
	)

	// swap out the default logger
	config.Default, _ = config.DefaultConfig.With(config.Stream(stream))
	log.Debugf("I can count 1 2 %d", 3)
	log.Infof("and more 4 5 %d", 6)

	// print what we logged
	fmt.Printf("%d\n", len(logs))
	for i := range logs {
		fmt.Println(logs[i])
	}

	// Output:
	// 1
	// Iand more 4 5 6
}
