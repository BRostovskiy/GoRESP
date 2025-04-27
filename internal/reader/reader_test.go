package reader

import (
	"bufio"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Split(splitter)
	return scanner
}

func TestRESP_readLine(t *testing.T) {
	type fields struct {
		Scanner *bufio.Scanner
	}

	tests := []struct {
		name    string
		fields  fields
		want    []byte
		want1   int
		wantErr bool
	}{
		{
			name: "test 1",
			fields: struct {
				Scanner *bufio.Scanner
			}{
				Scanner: getScanner(strings.NewReader("*3\r\n$3\r\nHEY\r\n$6\r\nMOTHER\r\n$6\r\nFUCKER")),
			},
			want: []byte("*3"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RESP{
				Scanner: tt.fields.Scanner,
			}
			got, err := r.readLine()
			assert.NoError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestRESP_Read(t *testing.T) {
	type fields struct {
		Scanner *bufio.Scanner
	}
	tests := []struct {
		name   string
		fields fields
		want   Value
	}{
		{
			name: "test1",
			fields: fields{
				Scanner: getScanner(strings.NewReader("*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n")),
			},
			want: Value{
				name: hrArray,
				content: []Value{
					{name: hrArray,
						content: []Value{
							{name: hrInteger, content: 1},
							{name: hrInteger, content: 2},
							{name: hrInteger, content: 3},
						}}, {
						name: hrArray,
						content: []Value{
							{name: hrString, content: "+Hello"},
							{name: hrError, content: "-World"},
						},
					},
				},
			},
		},
		{
			name: "test2",
			fields: fields{
				Scanner: getScanner(strings.NewReader("*3\r\n$3\r\nHEY\r\n$6\r\nMOTHER\r\n$3\r\nPUPPY")),
			},
			want: Value{
				name: hrArray,
				content: []Value{
					{
						name:    hrBulk,
						content: "HEY",
					},
					{
						name:    hrBulk,
						content: "MOTHER",
					},
					{
						name:    hrBulk,
						content: "PUPPY",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RESP{
				Scanner: tt.fields.Scanner,
			}
			got, err := r.Read()
			assert.NoError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}
