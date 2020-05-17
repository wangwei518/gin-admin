package entity

import (
	"context"
	"fmt"
	"time"

	"github.com/wangwei518/gin-admin/internal/app/config"
	"github.com/wangwei518/gin-admin/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)
type IndexModel struct {
	[]map[string]interface{}
	mappings	`json:"mappings"`
} 
type setting struct {
	name int,
	"number_of_replicas":0
}
type mappings struct {
	doc

}
type doc struct {
	properties
}

// Model base model
type Model struct {
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// IndexName collection name
func (Model) IndexName(name string) string {
	return fmt.Sprintf("%s%s", config.C.Elasticsearch.IndexPrefix, name)
}

// CreateIndexes 创建索引
func (Model) CreateIndexes(ctx context.Context, cli *elastic.Client, m indexer, settings map[string]interface{}, properties []IndexModel) error {

	mmp = make(map[string]interface{})

	mmp["username"] = "Murphy"
	mmp["age"] = 19
	mmp["sex"] = "man"

	models := []mongo.IndexModel{
		{Keys: bson.M{"created_at": 1}},
		{Keys: bson.M{"updated_at": 1}},
		{Keys: bson.M{"deleted_at": 1}},
	}
	if len(indexes) > 0 {
		models = append(models, indexes...)
	}
	_, err := getIndex(ctx, cli, m).Indexes().CreateMany(ctx, models)
	return err
}

func toString(v interface{}) string {
	return util.JSONMarshalToString(v)
}

type indexer interface {
	IndexName() string
}

func getIndex(ctx context.Context, cli *mongo.Client, m indexer) *mongo.Collection {
	return cli.Database(config.C.Mongo.Database).Collection(m.CollectionName())
}
