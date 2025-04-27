package server

import (
	"context"
	"fmt"

	"goredis/internal/reader"
)

const (
	// StopWord if client want to break the connection he basically needs to send this word
	getCmd  = "GET"
	setCmd  = "SET"
	stopCmd = "QUIT"
)

var (
	RootValueIsNotAnArrayError  = fmt.Errorf("root value is not an array")
	CouldNotGetHandlerNameError = fmt.Errorf("could not get command name")
	IncorrectArgsLengthError    = fmt.Errorf("incorrect number of argumens")
)

func get(ctx context.Context, storage Storage, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, IncorrectArgsLengthError
	}
	key, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("could not convert %v to string", args[0])
	}
	return storage.Get(ctx, key)
}

func set(ctx context.Context, storage Storage, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, IncorrectArgsLengthError
	}
	key, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("could not convert %v to string", args[0])
	}
	return nil, storage.Set(ctx, key, args[1])
}

func quit(ctx context.Context, _ Storage, _ ...interface{}) (interface{}, error) {
	return nil, nil
}

type Handler struct {
	name    string
	args    []interface{}
	Execute func(ctx context.Context, storage Storage, args ...interface{}) (interface{}, error)
}

func (h Handler) Args() []interface{} {
	return h.args
}

func (h Handler) IsStop() bool {
	return h.name == stopCmd
}

func NewHandlerFromValue(v reader.Value) (*Handler, error) {
	if v.IsArray() {
		array, ok := v.AsArray()
		if !ok {
			return nil, RootValueIsNotAnArrayError
		}

		cmdName, ok := array[0].String()
		if !ok {
			return nil, CouldNotGetHandlerNameError
		}

		l := len(array) - 1
		args := make([]interface{}, l)
		for i := 0; i < l; i++ {
			args[i] = array[i+1].Content()
		}

		cmd := &Handler{
			args: args,
		}
		switch cmdName {
		case getCmd:
			cmd.name = getCmd
			cmd.Execute = get
		case setCmd:
			cmd.name = setCmd
			cmd.Execute = set
		case stopCmd:
			cmd.name = cmdName
			cmd.Execute = quit
		default:
			return nil, fmt.Errorf("unsupported command: '%s' (use uppercase instead)", cmdName)
		}
		return cmd, nil
	}
	return nil, RootValueIsNotAnArrayError
}
