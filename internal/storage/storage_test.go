package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInMemoryStorage_GetSet(t *testing.T) {
	t.Parallel()
	l := logrus.New()

	storage := NewInMemory(l)
	storage.Run(context.Background())
	storage.data = map[string]interface{}{"hello": "world"}
	defer t.Cleanup(func() {
		storage.Done()
	})

	testCases := []struct {
		name    string
		key     string
		val     string
		want    string
		wantErr error
	}{
		{
			name:    "foo",
			key:     "world",
			wantErr: fmt.Errorf("[GET] key 'world' does not found"),
		},
		{
			name: "buzz",
			key:  "hello",
			want: "world",
		},
		{
			name: "GetSet",
			key:  "set",
			val:  "get",
			want: "get",
		},
	}
	for _, tc := range testCases {
		tc := tc
		// ^ tc variable reinitialised
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.val != "" {
				err := storage.Set(context.Background(), tc.key, tc.val)
				assert.NoError(t, err)
			}
			got, err := storage.Get(context.Background(), tc.key)
			if tc.wantErr != nil {
				assert.Error(t, err, tc.wantErr)
			} else {
				assert.Equal(t, got, tc.want)
				assert.NoError(t, err)
			}
		})
	}
}
