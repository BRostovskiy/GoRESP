package reader

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

var (
	separator = []byte("\r\n")
)

const (
	respSimpleString byte = '+'
	respError        byte = '-'
	respInteger           = ':'
	respBulkString   byte = '$'
	respArray        byte = '*'

	hrInteger = "int"
	hrBulk    = "bulk"
	hrArray   = "array"
	hrString  = "str"
	hrError   = "err"
)

func splitter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, separator); i >= 0 {
		return i + 2, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return
}

type Value struct {
	name    string
	content interface{}
}

func (v *Value) AsArray() ([]Value, bool) {
	result, ok := v.content.([]Value)
	return result, ok
}

func (v *Value) IsArray() bool {
	return v.name == hrArray
}

func (v *Value) String() (string, bool) {
	result, ok := v.content.(string)
	return result, ok
}

func (v *Value) Content() interface{} {
	return v.content
}

type RESP struct {
	*bufio.Scanner
}

func NewRESP(reader io.Reader) *RESP {
	r := RESP{Scanner: bufio.NewScanner(reader)}
	r.Split(splitter)
	return &r
}

func (r *RESP) Read() (Value, error) {
	line, err := r.readLine()
	if err != nil {
		return Value{}, err
	}

	switch line[0] {
	case respArray:
		arrayLen, err := r.toInteger(line[1:])
		if err != nil {
			return Value{}, err
		}
		return r.readArray(arrayLen)
	case respBulkString:
		arrayLen, err := r.toInteger(line[1:])
		if err != nil {
			return Value{}, err
		}
		return r.readBulk(arrayLen)
	case respSimpleString:
		return Value{
			name:    hrString,
			content: string(line),
		}, nil
	case respInteger:
		i, err := r.toInteger(line[1:])
		if err != nil {
			return Value{}, err
		}

		return Value{
			name:    hrInteger,
			content: i,
		}, nil
	case respError:
		return Value{
			name:    hrError,
			content: string(line),
		}, err
	default:
		return Value{}, fmt.Errorf("unknown type %s", string(line[0]))
	}
}

func (r *RESP) toInteger(byteInt []byte) (int, error) {
	i64, err := strconv.ParseInt(string(byteInt), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not convert to integer: %w", err)
	}
	return int(i64), nil
}

func (r *RESP) readLine() ([]byte, error) {
	if r.Scan() {
		line := r.Text()
		return []byte(line), nil
	}
	return nil, io.EOF
}

func (r *RESP) readArray(l int) (Value, error) {
	v := Value{
		name: hrArray,
	}

	content := make([]Value, l)

	for i := 0; i < l; i++ {
		val, err := r.Read()
		if err != nil {
			return v, err
		}
		content[i] = val
	}

	v.content = content
	return v, nil
}

func (r *RESP) readBulk(length int) (Value, error) {
	v := Value{
		name: hrBulk,
	}

	// null string
	if length == -1 {
		return Value{name: hrBulk, content: "$"}, nil
	}

	// empty string
	if length == 0 {
		return Value{name: hrString, content: "+"}, nil
	}

	line, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	v.content = string(line)

	return v, nil
}
