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
	"fmt"

	"github.com/jdef/log"
	"github.com/jdef/log/config"
)

func Example_withCustomLogger() {
	var (
		logs   = []string{}
		logger = config.LogFunc(func(m string, a ...interface{}) {
			logs = append(logs, fmt.Sprintf(m, a...))
		})
	)

	// swap out the default log sink
	config.Default, _ = config.DefaultConfig.With(config.Sink(logger))
	log.Debugf("I can count 1 2 %d", 3)
	log.Infof("and more 4 5 %d", 6)

	// print what we logged
	fmt.Printf("%d\n", len(logs))
	fmt.Print(logs[0])

	// Output:
	// 1
	// and more 4 5 6
}
