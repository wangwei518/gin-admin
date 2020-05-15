package jwtauth

import (
	"context"
	"testing"

	//"github.com/wangwei518/gin-admin/pkg/auth/jwtauth/store/buntdb"
	"github.com/stretchr/testify/assert"
	"github.com/wangwei518/gin-admin/pkg/auth/jwtauth/store/elasticsearch"
)

func TestAuth(t *testing.T) {
	//store, err := buntdb.NewStore(":memory:")
	store, err := elasticsearch.NewStore(&elasticsearch.Config{
		URL:      "http://127.0.0.1:9200",
		User:     "elastic",
		Password: "123456",
		Index:    "ga_" + "auth_blacklist",
	})
	assert.Nil(t, err)

	jwtAuth := New(store)

	defer jwtAuth.Release()

	ctx := context.Background()
	userID := "test"
	userView := "global"
	token, err := jwtAuth.GenerateToken(ctx, userID, userView)
	assert.Nil(t, err)
	assert.NotNil(t, token)

	id, view, err := jwtAuth.ParseUserID(ctx, token.GetAccessToken())
	assert.Nil(t, err)
	assert.Equal(t, userID, id)
	assert.Equal(t, userView, view)

	err = jwtAuth.DestroyToken(ctx, token.GetAccessToken())
	assert.Nil(t, err)

	id, view, err = jwtAuth.ParseUserID(ctx, token.GetAccessToken())
	assert.NotNil(t, err)
	assert.EqualError(t, err, "invalid token")
	assert.Empty(t, id)
}
