// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package main

import (
	"context"
	"github.com/harness-community/drone-coverage-report/plugin"

	pd "github.com/harness-community/drone-coverage-report/plugin/plugin_defs"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(new(formatter))

	var args pd.Args
	if err := envconfig.Process("", &args); err != nil {
		logrus.Fatalln(err)
	}

	switch args.Level {
	case "debug":
		logrus.SetFormatter(textFormatter)
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetFormatter(textFormatter)
		logrus.SetLevel(logrus.TraceLevel)
	}

	if _, err := plugin.Exec(context.Background(), args); err != nil {
		logrus.Fatalln(err)
	}
}

// default formatter that writes logs without including timestamp
// or level information.
type formatter struct{}

func (*formatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message), nil
}

// text formatter that writes logs with level information
var textFormatter = &logrus.TextFormatter{
	DisableTimestamp: true,
}
