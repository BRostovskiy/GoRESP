package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"goredis/internal/reader"
)

type Storage interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, val interface{}) error
}

// NewTCP constructor for ServerTCP structure
func NewTCP(storage Storage, log *logrus.Logger) *TCP {
	return &TCP{
		mu:      sync.Mutex{},
		done:    make(chan struct{}),
		storage: storage,
		log:     log,
	}
}

// TCP basic structure for TCP resp-server
type TCP struct {
	done        chan struct{}
	mu          sync.Mutex
	storage     Storage
	log         *logrus.Logger
	ConnContext func(ctx context.Context, c net.Conn) context.Context
}

// Done close done channel for tcp and force hit to shut down
func (srv *TCP) Done() {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	select {
	case <-srv.done:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(srv.done)
	}
}

// ListenAndServe main loop of handling ongoing connections
func (srv *TCP) ListenAndServe(address string) error {
	s, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed listen address %s: %w", address, err)
	}

	srv.log.Debugf("Bind and ready %s!", address)

	connsChan := clientConns(srv.log, s)
	for {
		select {
		case <-srv.done:
			return http.ErrServerClosed
		case conn := <-connsChan:
			connCtx := context.Background()
			if cc := srv.ConnContext; cc != nil {
				connCtx = cc(connCtx, conn)
				if connCtx == nil {
					panic("ConnContext returned nil")
				}
			}
			go handleConnection(connCtx, srv.log, conn, srv.storage)
		}

	}
}

func clientConns(log *logrus.Logger, listener net.Listener) chan net.Conn {
	ch := make(chan net.Conn)
	go func() {
		for {
			client, err := listener.Accept()
			if client == nil {
				log.Errorf("couldn't accept: %v", err)
				continue
			}
			ch <- client
		}
	}()
	return ch
}

func handleConnection(ctx context.Context, log *logrus.Logger, conn net.Conn, storage Storage) {
	defer func() { _ = conn.Close() }()
	log.Debugf("new connection: %v <-> %v", conn.LocalAddr(), conn.RemoteAddr())

	resp := reader.NewRESP(conn)
	for {
		value, err := resp.Read()
		if err != nil {
			if err != io.EOF {
				log.Errorf("error during read: %v", err)
				writeError(conn, err)
			}
			return
		}

		h, err := NewHandlerFromValue(value)
		if err != nil {
			writeError(conn, err)
			log.Errorf("error during getting handler: %v", err)
			return
		}

		res, err := h.Execute(ctx, storage, h.Args()...)
		if err != nil {
			log.Errorf("error during execute: %v", err)
			writeError(conn, err)
			continue
		}

		if res != nil {
			log.Debugf("result: %v", res)
			writeString(conn, res.(string))
			continue
		}

		writeString(conn, "OK")
		if h.IsStop() {
			return
		}
	}
}

func writeString(w io.Writer, str string) {
	_, _ = fmt.Fprintf(w, "+%s\r\n", str)
}

func writeError(w io.Writer, err error) {
	_, _ = fmt.Fprintf(w, "-ERR: %v\r\n", err)
}
