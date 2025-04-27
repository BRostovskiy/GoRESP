package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"goredis/internal/server"
	"goredis/internal/storage"

	"github.com/namsral/flag"
	"github.com/sirupsen/logrus"
)

type config struct {
	BindAddr string
	BindPort int
	LogLevel string
}

func waitForShutdown(log *logrus.Logger, storage runAndDone, h *server.TCP) {
	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-term:
		log.Debugf("terminating by signal %d", sig)
		storage.Done()
		h.Done()
		time.Sleep(1 * time.Second)
	}
}

func initLogger(level string) *logrus.Logger {
	l := logrus.New()

	switch strings.ToLower(level) {
	case "debug":
		l.Level = logrus.DebugLevel
	case "error":
		l.Level = logrus.ErrorLevel
	case "warn":
		l.Level = logrus.WarnLevel
	default:
		l.Level = logrus.InfoLevel
	}

	return l
}

func parseFlags() config {
	var cfg config
	flags := flag.NewFlagSet("GoRedis", flag.ContinueOnError)
	flags.StringVar(&cfg.LogLevel, "log-level", "info",
		"Log level. Available options: debug, info, warn, error")
	flags.StringVar(&cfg.BindAddr, "bind_addr", "", "bind_addr=<IP>")
	flags.IntVar(&cfg.BindPort, "bind_port", 6379, "bind_port=<INT>")
	flags.SetOutput(io.Discard)
	err := flags.Parse(os.Args[1:])

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Printf("GoRedis\n\n")
			fmt.Printf("USAGE\n\n  %s [OPTIONS]\n\n", os.Args[0])
			fmt.Print("OPTIONS\n\n")
			flags.SetOutput(os.Stdout)
			flags.PrintDefaults()
			os.Exit(0)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	return cfg
}

type runAndDone interface {
	Run(ctx context.Context)
	Done()
}

func run(cfg config, storage runAndDone, handler *server.TCP) {
	storage.Run(context.Background())
	go func() {
		err := handler.ListenAndServe(fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.BindPort))
		if err != nil {
			log.Fatalf("%v", err)
			return
		}
	}()
}

func main() {

	cfg := parseFlags()

	logger := initLogger(cfg.LogLevel)

	s := storage.NewInMemory(logger)
	handler := server.NewTCP(s, logger)
	run(cfg, s, handler)
	waitForShutdown(logger, s, handler)
}
