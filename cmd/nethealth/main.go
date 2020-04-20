/*
Copyright 2019 Gravitational, Inc.

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

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gravitational/satellite/lib/nethealth"

	"github.com/gravitational/trace"
	"github.com/gravitational/version"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

func init() {
	version.Init("v0.0.1-master+$Format:%h$")
}

func main() {
	if err := run(); err != nil {
		log.Fatal(trace.DebugReport(err))
	}
}

func run() error {
	var (
		app   = kingpin.New("satellite", "Cluster health monitoring agent")
		debug = app.Flag("debug", "Enable verbose mode").Bool()

		// `version` command
		cversion = app.Command("version", "Display version")

		// `run` command
		crun               = app.Command("run", "Start nethealth agent")
		crunPrometheusPort = crun.Flag("prom-port", "The prometheus port to bind to").Default("9801").Uint32()
		crunNamespace      = crun.Flag("namespace", "The kubernetes namespace to watch for nethealth pods").
					Default("monitoring").OverrideDefaultFromEnvar("POD_NAMESPACE").String()
		crunHostIP   = crun.Flag("host-ip", "The host IP address").OverrideDefaultFromEnvar("HOST_IP").String()
		crunSelector = crun.Flag("pod-selector", "The kubernetes selector to identify nethealth pods").
				Default(nethealth.DefaultSelector).String()
	)

	var cmd string
	var err error

	cmd, err = app.Parse(os.Args[1:])
	if err != nil {
		return trace.Errorf("unable to parse command line.\nUse nethealth --help for help.")
	}

	log.SetOutput(os.Stderr)
	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = time.RFC3339Nano
	log.SetFormatter(customFormatter)

	switch cmd {
	case cversion.FullCommand():
		version.Print()
		return nil

	case crun.FullCommand():
		appConfig := nethealth.AppConfig{
			PrometheusPort: *crunPrometheusPort,
			Namespace:      *crunNamespace,
			HostIP:         *crunHostIP,
			Selector:       *crunSelector,
		}

		app, err := nethealth.NewApp(appConfig)
		if err != nil {
			return trace.Wrap(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigs
			cancel()
			log.Info("Exiting on signal: ", sig)
		}()

		app.Run(ctx)
	}

	return nil
}
