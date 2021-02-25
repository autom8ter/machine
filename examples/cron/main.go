package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/autom8ter/machine/v2"
	"github.com/autom8ter/machine/v2/examples/helpers"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func init() {
	flag.StringVar(&configPath, "config", "cron.yaml", "path to cron config file")
	flag.IntVar(&port, "port", 8000, "port to serve http on")
	flag.Parse()
}

var (
	configPath string
	port       int
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
	var server *http.Server
	m := machine.New(machine.WithErrHandler(func(err error) {
		if errors.Unwrap(err) == http.ErrServerClosed && server != nil {
			server.Close()
		}
	}))
	mux := http.NewServeMux()
	mux.HandleFunc("/cron/job", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST method", http.StatusBadRequest)
			logger.Error("expected POST method", zap.String("method", r.Method))
			return
		}
		var j job
		if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
			http.Error(w, "failed to decode job", http.StatusBadRequest)
			logger.Error("failed to decode job", zap.Error(err))
			return
		}
		dur, err := time.ParseDuration(j.Sleep)
		if err != nil {
			http.Error(w, "failed to parse job duration", http.StatusBadRequest)
			logger.Error("failed to parse job duration",
				zap.Error(err),
				zap.String("sleep", j.Sleep),
				zap.String("name", j.Name),
				zap.String("script", j.Script),
			)
			return
		}
		c.Jobs = append(c.Jobs, &j)
		m.Cron(ctx, dur, func(ctx context.Context) (bool, error) {
			execJob(ctx, logger, &j)
			return true, nil
		})
	})
	mux.HandleFunc("/cron", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "expected GET method", http.StatusBadRequest)
			logger.Error("expected GET method", zap.String("method", r.Method))
			return
		}
		json.NewEncoder(w).Encode(c)
	})
	server = &http.Server{
		Handler: mux,
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		logger.Error("failed to create listener", zap.Error(err))
		return
	}

	m.Go(ctx, func(ctx context.Context) error {
		if err := server.Serve(lis); err != nil && err != http.ErrServerClosed {
			return errors.Wrap(err, "server failure")
		}
		return nil
	})
	m.Go(ctx, func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				logger.Info("shutting down server!")
				server.Close()
				return lis.Close()
			}
		}
	})

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
		m.Cron(ctx, dur, func(ctx context.Context) (bool, error) {
			execJob(ctx, logger, j)
			return true, nil
		})
	}
	time.Sleep(1 * time.Second)
	m.Wait()
	logger.Info("shutting down...")
}

func execJob(ctx context.Context, logger *zap.Logger, j *job) {
	logger.Info("executing job",
		zap.String("name", j.Name),
		zap.String("sleep", j.Sleep),
		zap.String("script", j.Script),
	)
	out := shell(ctx, j.Script)
	logger.Info("command finished",
		zap.String("name", j.Name),
		zap.String("sleep", j.Sleep),
		zap.String("script", j.Script),
		zap.String("output", out),
	)
}

type job struct {
	Name   string `yaml:"name" json:"name"`
	Script string `yaml:"script" json:"script"`
	Sleep  string `yaml:"sleep" json:"sleep"`
}

type cron struct {
	Name string            `yaml:"name" json:"name"`
	Env  map[string]string `yaml:"env" json:"env"`
	Jobs []*job            `yaml:"jobs" json:"jobs"`
}

func shell(ctx context.Context, script string) string {
	e := exec.CommandContext(ctx, "/bin/bash", "-c", script)
	e.Env = os.Environ()
	res, _ := e.CombinedOutput()
	return strings.TrimSpace(string(res))
}
