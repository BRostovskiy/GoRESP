package handler

import (
	"fmt"

	"goredis/internal/reader"
	"goredis/internal/repo"
)

const (
	// StopWord if client want to break the connection he basically needs to send this word
	getCmd     = "GET"
	setCmd     = "SET"
	stopCmd    = "QUIT"
	clientCmd  = "CLIENT"
	commandCmd = "COMMAND"
)

var (
	RootValueIsNotAnArrayError  = fmt.Errorf("root value is not an array")
	CouldNotGetHandlerNameError = fmt.Errorf("could not get command name")
	IncorrectArgsLengthError    = fmt.Errorf("incorrect number of argumens")
)

func Get(storage repo.KVStorage, args ...interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, IncorrectArgsLengthError
	}
	key, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("could not convert %v to string", args[0])
	}
	return storage.Get(key)
}

func Set(storage repo.KVStorage, args ...interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, IncorrectArgsLengthError
	}
	key, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("could not convert %v to string", args[0])
	}
	return nil, storage.Set(key, args[1])
}

func dummy(_ repo.KVStorage, _ ...interface{}) (interface{}, error) {
	return nil, nil
}

type Handler struct {
	name    string
	args    []interface{}
	Execute func(storage repo.KVStorage, args ...interface{}) (interface{}, error)
}

func (h Handler) Args() []interface{} {
	return h.args
}

func (h Handler) IsStop() bool {
	return h.name == stopCmd
}

func FromValue(v reader.Value) (*Handler, error) {
	if v.IsArray() {
		array, ok := v.AsArray()
		if !ok {
			return nil, RootValueIsNotAnArrayError
		}

		cmdName, ok := array[0].AsString()
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
			cmd.Execute = Get
		case setCmd:
			cmd.name = setCmd
			cmd.Execute = Set
		case stopCmd, commandCmd, clientCmd:
			cmd.name = cmdName
			cmd.Execute = dummy
		default:
			return nil, fmt.Errorf("unsupported command: '%s' (use uppercase instead)", cmdName)
		}
		return cmd, nil
	}
	return nil, RootValueIsNotAnArrayError
}
