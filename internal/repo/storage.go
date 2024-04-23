package repo

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	getCmd = "get"
	setCmd = "set"
)

type KVStorage interface {
	Start(ctx context.Context)
	Get(key string) (interface{}, error)
	Set(key string, val interface{}) error
	Done()
}

type InMemoryStorage struct {
	data map[string]interface{}
	in   chan command
	mu   sync.Mutex
	done chan struct{}
	log  *logrus.Logger
}

func NewInMemoryStorage(log *logrus.Logger) *InMemoryStorage {
	return &InMemoryStorage{
		data: make(map[string]interface{}),
		in:   make(chan command, 1),
		mu:   sync.Mutex{},
		done: make(chan struct{}),
		log:  log,
	}
}

type ourRecord [2]interface{}
type outChannel chan ourRecord

type command struct {
	Cmd  string
	Args []interface{}
	Out  outChannel
}

func (ims *InMemoryStorage) processGet(_ context.Context, out outChannel, args []interface{}) {
	if len(args) != 1 {
		out <- ourRecord{nil, fmt.Errorf("[GET] not suficiend arguments")}
		return
	}

	key, isString := args[0].(string)
	if !isString {
		out <- ourRecord{nil, fmt.Errorf("[GET] could not convert key")}
		return
	}
	val, ok := ims.data[key]
	if !ok {
		out <- ourRecord{nil, fmt.Errorf("[GET] key '%s' does not found", key)}
		return
	}
	out <- ourRecord{val, nil}
}

func (ims *InMemoryStorage) processSet(_ context.Context, out outChannel, args []interface{}) {
	if len(args) != 2 {
		out <- ourRecord{nil, fmt.Errorf("[SET] not suficiend arguments")}
		return
	}
	key, isString := args[0].(string)
	if !isString {
		out <- ourRecord{nil, fmt.Errorf("[SET] could not convert key")}
		return
	}
	ims.data[key] = args[1]
	out <- ourRecord{nil, nil}
}

func (ims *InMemoryStorage) Done() {
	ims.mu.Lock()
	defer ims.mu.Unlock()

	select {
	case <-ims.done:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(ims.in)
		close(ims.done)
	}
}

func (ims *InMemoryStorage) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				ims.log.Debugf("storage done by the context")
				ims.Done()
				return
			case in := <-ims.in:
				switch in.Cmd {
				case getCmd:
					ims.processGet(context.Background(), in.Out, in.Args)
				case setCmd:
					ims.processSet(context.Background(), in.Out, in.Args)
				}
			}

		}
	}()
}

func (ims *InMemoryStorage) Get(key string) (interface{}, error) {
	out := make(chan ourRecord)
	defer close(out)

	select {
	case <-ims.done:
		return nil, nil
	default:
		ims.in <- command{
			Cmd:  getCmd,
			Args: []interface{}{key},
			Out:  out,
		}
		res := <-out
		if res[1] != nil {
			return nil, res[1].(error)
		}
		return res[0], nil
	}
}

func (ims *InMemoryStorage) Set(key string, val interface{}) error {
	out := make(chan ourRecord)
	defer close(out)

	select {
	case <-ims.done:
		return nil
	default:
		ims.in <- command{
			Cmd:  setCmd,
			Args: []interface{}{key, val},
			Out:  out,
		}
		res := <-out
		if res[1] == nil {
			return nil
		}
		return res[1].(error)
	}
}
