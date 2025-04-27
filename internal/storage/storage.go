package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type (
	action string
)

const (
	getCmd action = "get"
	setCmd action = "set"
)

type InMemory struct {
	data map[string]interface{}
	in   chan command
	mu   sync.Mutex
	done chan struct{}
	log  *logrus.Logger
}

func NewInMemory(log *logrus.Logger) *InMemory {
	return &InMemory{
		data: make(map[string]interface{}),
		in:   make(chan command, 1),
		mu:   sync.Mutex{},
		done: make(chan struct{}),
		log:  log,
	}
}

type outRecord [2]interface{}
type outChannel chan outRecord

type command struct {
	Cmd  action
	Args []interface{}
	Out  outChannel
}

func (ims *InMemory) get(output outChannel, args []interface{}) {
	if len(args) != 1 {
		output <- outRecord{nil, fmt.Errorf("[GET] not suficiend arguments")}
		return
	}

	key, isString := args[0].(string)
	if !isString {
		output <- outRecord{nil, fmt.Errorf("[GET] could not convert key")}
		return
	}
	val, ok := ims.data[key]
	if !ok {
		output <- outRecord{nil, fmt.Errorf("[GET] key '%s' does not found", key)}
		return
	}
	output <- outRecord{val, nil}
}

func (ims *InMemory) set(out outChannel, args []interface{}) {
	if len(args) != 2 {
		out <- outRecord{nil, fmt.Errorf("[SET] not suficiend arguments")}
		return
	}
	key, isString := args[0].(string)
	if !isString {
		out <- outRecord{nil, fmt.Errorf("[SET] could not convert key")}
		return
	}
	ims.data[key] = args[1]
	out <- outRecord{nil, nil}
}

func (ims *InMemory) Done() {
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

func (ims *InMemory) Run(ctx context.Context) {
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
					ims.get(in.Out, in.Args)
				case setCmd:
					ims.set(in.Out, in.Args)
				}
			}

		}
	}()
}

func (ims *InMemory) Get(ctx context.Context, key string) (interface{}, error) {
	out, err := call(ctx, getCmd, ims.in, key)
	if err != nil {
		return nil, err
	}
	return out[0], nil
}

func (ims *InMemory) Set(ctx context.Context, key string, val interface{}) error {
	_, err := call(ctx, setCmd, ims.in, key, val)
	if err != nil {
		return err
	}
	return nil
}

func call(ctx context.Context, cmd action, input chan command, args ...interface{}) (*outRecord, error) {
	out := make(chan outRecord)
	defer close(out)

	select {
	case <-ctx.Done():
		return nil, nil
	default:
		input <- command{
			Cmd:  cmd,
			Args: args,
			Out:  out,
		}
		res := <-out
		if res[1] != nil {
			return nil, res[1].(error)
		}
		return &outRecord{res[0]}, nil
	}
}
