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
	"os"
	"path/filepath"
	"strings"

	"github.com/gologs/log"
	"github.com/gologs/log/caller"
	"github.com/gologs/log/config"
	"github.com/gologs/log/context"
	"github.com/gologs/log/encoding"
	"github.com/gologs/log/io"
	"github.com/gologs/log/io/ioutil"
	"github.com/gologs/log/levels"
	"github.com/gologs/log/logger"
	"github.com/gologs/log/logger/redact"
)

func Example_withCustomLogger() {
	var (
		logs    = []string{}
		flogger = logger.Func(func(_ context.Context, m string, a ...interface{}) {
			if m == "" {
				logs = append(logs, fmt.Sprint(a...))
			} else {
				logs = append(logs, fmt.Sprintf(m, a...))
			}
		})
	)

	// swap out the default logger
	config.Logging = config.DefaultConfig.With(config.Logger(flogger))
	log.Debugf("I can count 1 2 %d", 3)
	log.Logf("and more 4 5 %d", 6)

	govetIgnoreFormatString := func() string { return "7 %%" }
	log.Log(govetIgnoreFormatString(), 8, 9)

	// print what we logged
	fmt.Printf("%d\n", len(logs))
	for i := range logs {
		fmt.Println(logs[i])
	}

	// Output:
	// 2
	// and more 4 5 6
	// 7 %%8 9
}

func Example_withCustomStream() {
	var (
		logs   = []string{}
		stream = &io.BufferedStream{
			EOMFunc: func(b io.Buffer, err error) error {
				if err != nil {
					return err
				}
				logs = append(logs, b.String())
				return nil
			},
		}
	)

	// swap out the default logger
	config.Logging = config.DefaultConfig.With(
		config.OnPanic(config.NoPanic()),
		config.OnExit(config.NoExit()),
		config.Stream(stream),
		config.Encoding(ioutil.LevelPrefix()),
	)
	log.Debugf("I can count 1 2 %d", 3)
	log.Infof("and more 4 5 %d", 6)
	log.Warnf("and more 5 6 %d", 7)
	log.Errorf("and more 6 7 %d", 8)
	log.Fatalf("and more 7 8 %d", 9)
	log.Panicf("and more 8 9 %d", 0)

	log.Debug("debug w/o", "format")
	log.Info("info w/o", "format")
	log.Warn("warn w/o", "format")
	log.Error("error w/o", "format")
	log.Fatal("fatal w/o", "format")
	log.Panic("panic w/o", "format")

	// print what we logged
	fmt.Printf("%d\n", len(logs))
	for i := range logs {
		fmt.Println(logs[i])
	}

	// Output:
	// 10
	// Iand more 4 5 6
	// Wand more 5 6 7
	// Eand more 6 7 8
	// Fand more 7 8 9
	// Pand more 8 9 0
	// Iinfo w/oformat
	// Wwarn w/oformat
	// Eerror w/oformat
	// Ffatal w/oformat
	// Ppanic w/oformat
}

func Example_withCustomMarshaler() {
	var (
		logs   = []string{}
		stream = &io.BufferedStream{
			EOMFunc: func(b io.Buffer, err error) error {
				if err != nil {
					return err
				}
				logs = append(logs, b.String())
				return nil
			},
		}
		// key=value marshaler
		marshaler = func(ctx context.Context, w io.Stream, m string, a ...interface{}) (err error) {
			caller, ok := caller.FromContext(ctx)
			if ok {
				a = append(a,
					"file", filepath.Base(caller.File),
					"line", caller.Line,
					"func", caller.FuncName[strings.LastIndexByte(caller.FuncName, byte('.'))+1:])
			}
			fmt.Fprint(w, m)
			_, _ = w.Write([]byte("{"))
			if len(a) > 0 {
				for i := 0; i+1 < len(a); i++ {
					if i > 0 {
						_, _ = w.Write([]byte(","))
					}
					fmt.Fprint(w, a[i])
					_, _ = w.Write([]byte("="))
					i++
					fmt.Fprint(w, a[i])
				}
			}
			_, _ = w.Write([]byte("}"))
			return w.EOM(nil)
		}
	)

	// swap out the default logger
	config.Logging = config.DefaultConfig.With(
		config.OnPanic(config.NoPanic()),
		config.OnExit(config.NoExit()),
		config.Stream(stream),
		config.Marshaler(marshaler),
		config.Encoding(ioutil.LevelPrefix()),
	)
	log.Info("k%", "v", "majorVersion", 1, "module", "storage")

	// print what we logged
	fmt.Printf("%d\n", len(logs))
	for i := range logs {
		fmt.Println(logs[i])
	}

	// Output:
	// 1
	// I{k%=v,majorVersion=1,module=storage,file=log_test.go,line=171,func=Example_withCustomMarshaler}
}

type password struct {
	redact.Simple
	secret string
}

type creditcard struct {
	redact.Interface
	account string
}

func (p *password) String() string { return p.secret }

func newCreditCard(account string) creditcard {
	return creditcard{redact.Blackout(account), account}
}

func Example_withTextStream() {
	// illustates how to inject a logger.Decorator while making use of a custom stream
	log := config.DefaultConfig.With(
		config.Stream(io.TextStream(os.Stdout)),
		config.Encoding(ioutil.LevelPrefix()),
		config.Level(levels.Debug),
		config.Builder(func(s io.Stream, m encoding.Marshaler, e chan<- error) logger.Logger {
			return redact.Default(logger.WithStream(s, m, e))
		}))

	log.Debugf("password=%v", &password{secret: "mysecret"})
	log.Debugf("cc=%v", newCreditCard("1234-5678-9012-3456"))

	// Output:
	// Dpassword=xxREDACTEDxx
	// Dcc=xxxxxxxxxxxxxxxxxxx
}
