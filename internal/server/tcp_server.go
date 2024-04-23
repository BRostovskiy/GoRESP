package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"goredis/internal/handler"

	"goredis/internal/reader"
	"goredis/internal/repo"
)

// NewCommandServerTCP constructor for ServerTCP structure
func NewCommandServerTCP(storage repo.KVStorage, log *logrus.Logger) *TCP {
	return &TCP{
		mu:      sync.Mutex{},
		done:    make(chan struct{}),
		storage: storage,
		log:     log,
	}
}

// TCP basic structure for TCP resp-server
type TCP struct {
	done    chan struct{}
	mu      sync.Mutex
	storage repo.KVStorage
	log     *logrus.Logger
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
			go handleConnection(srv.log, conn, srv.storage)
		}

	}
}

func clientConns(log *logrus.Logger, listener net.Listener) chan net.Conn {
	ch := make(chan net.Conn)
	i := 0
	go func() {
		for {
			client, err := listener.Accept()
			if client == nil {
				log.Errorf("couldn't accept: %v", err)
				continue
			}
			i++
			ch <- client
		}
	}()
	return ch
}

func handleConnection(log *logrus.Logger, conn net.Conn, storage repo.KVStorage) {
	defer func() { _ = conn.Close() }()

	for {
		log.Debugf("new connection: %v <-> %v", conn.LocalAddr(), conn.RemoteAddr())

		resp := reader.NewRESPReader(conn)
		value, err := resp.Read()
		if err != nil {
			if err != io.EOF {
				log.Errorf("error during read: %v", err)
				writeError(conn, err)
			}
			return
		}

		h, err := handler.FromValue(value)
		if err != nil {
			writeError(conn, err)
			log.Errorf("error during getting handler: %v", err)
			return
		}

		res, err := h.Execute(storage, h.Args()...)
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
