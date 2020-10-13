package main

import (
	"context"
	"flag"
	"github.com/autom8ter/machine"
	"github.com/autom8ter/machine/examples/helpers"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

func init() {
	flag.StringVar(&configPath, "config", "cron.yaml", "path to cron config file")
	flag.Parse()
}

var (
	configPath string
)

func main() {
	logger := helpers.Logger(zap.String("example", "cron"))
	bits, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Error("failed to read config file", zap.Error(err))
		return
	}
	var c = &cron{}
	if err := yaml.Unmarshal(bits, c); err != nil {
		logger.Error("unmarshal config file", zap.Error(err))
		return
	}
	if len(c.Jobs) == 0 {
		logger.Error("zero jobs in cofig", zap.String("path", configPath))
		return
	}
	for k, v := range c.Env {
		os.Setenv(k, v)
	}
	m := machine.New(context.Background(),
		machine.WithTags([]string{c.Name}),
		machine.WithMiddlewares(machine.PanicRecover()),
	)
	for _, jb := range c.Jobs {
		dur, err := time.ParseDuration(jb.Sleep)
		if err != nil {
			logger.Error("failed to parse cron sleep duration",
				zap.Error(err),
				zap.String("sleep", jb.Sleep),
				zap.String("name", jb.Name),
				zap.String("script", jb.Script),
			)
			continue
		}
		j := jb //create local copy
		m.Go(func(routine machine.Routine) {
			logger.Info("executing job",
				zap.String("name", j.Name),
				zap.String("sleep", j.Sleep),
				zap.String("script", j.Script),
			)
			out := shell(routine.Context(), j.Script)
			logger.Info("command finished",
				zap.String("name", j.Name),
				zap.String("sleep", j.Sleep),
				zap.String("script", j.Script),
				zap.String("output", out),
			)
		}, machine.GoWithMiddlewares(machine.Cron(time.NewTicker(dur))))
	}
	time.Sleep(1 * time.Second)
	m.Wait()
	logger.Info("shutting down...")
}

type job struct {
	Name   string `yaml:"name"`
	Script string `yaml:"script"`
	Sleep  string `yaml:"sleep"`
}

type cron struct {
	Name string            `json:"name"`
	Env  map[string]string `yaml:"env"`
	Jobs []*job            `yaml:"jobs"`
}

func shell(ctx context.Context, script string) string {
	e := exec.CommandContext(ctx, "/bin/sh", "-c", script)
	e.Env = os.Environ()
	res, _ := e.CombinedOutput()
	return strings.TrimSpace(string(res))
}
