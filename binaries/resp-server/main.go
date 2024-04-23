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

	"goredis/internal/repo"
	"goredis/internal/server"

	"github.com/namsral/flag"
	"github.com/sirupsen/logrus"
)

type config struct {
	BindAddr string
	BindPort int
	LogLevel string
}

func startTCP(cfg *config, storage repo.KVStorage, logger *logrus.Logger) *server.TCP {
	h := server.NewCommandServerTCP(storage, logger)

	go func() {
		err := h.ListenAndServe(fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.BindPort))
		if err != nil {
			log.Fatalf("%v", err)
			return
		}
	}()

	return h
}

func waitForShutdown(log *logrus.Logger, storage repo.KVStorage, h *server.TCP) {
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

func main() {
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

	logger := initLogger(cfg.LogLevel)

	s := repo.NewInMemoryStorage(logger)
	s.Start(context.Background())
	h := startTCP(&cfg, s, logger)

	waitForShutdown(logger, s, h)
}
