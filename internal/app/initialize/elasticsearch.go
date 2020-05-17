package initialize

import (
	"context"

	"github.com/wangwei518/gin-admin/internal/app/config"
	ielastic "github.com/wangwei518/gin-admin/internal/app/model/impl/elasticsearch"

	"github.com/olivere/elastic/v7"
)

// InitElasticsearch ...
func InitElasticsearch() (*elastic.Client, func(), error) {

	cfg := config.C.Elasticsearch

	client, cleanFunc, err := ielastic.NewClient(&ielastic.Config{
		URL:      cfg.URL,
		User:     cfg.User,
		Password: cfg.Password,
		Index:    cfg.IndexPrefix,
	})
	if err != nil {
		return nil, cleanFunc, err
	}

	err = ielastic.CreateIndexes(context.Background(), client)
	if err != nil {
		return nil, cleanFunc, err
	}

	return client, cleanFunc, nil
}
