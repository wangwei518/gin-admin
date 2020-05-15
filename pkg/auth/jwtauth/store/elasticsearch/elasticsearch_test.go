package elasticsearch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	addr = "127.0.0.1:6379"
)

func TestStore(t *testing.T) {
	store, err := NewStore(&Config{
		URL:      "http://127.0.0.1:9200",
		User:     "elastic",
		Password: "123456",
		Index:    "ga_" + "auth_blacklist",
	})
	if err != nil {
		panic(err)
	}

	defer store.Close()

	key := "1234567890.ABCDEFGHIJKLMNOPQRSTUVWXYZ.!@#$%^&*()"
	//key := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	//key := "ABCDE"
	ctx := context.Background()

	err = store.Set(ctx, key, 0)
	assert.Nil(t, err)

	b, err := store.Check(ctx, key)
	assert.Nil(t, err)
	assert.Equal(t, true, b)

	err = store.Delete(ctx, key)
	assert.Nil(t, err)
}
