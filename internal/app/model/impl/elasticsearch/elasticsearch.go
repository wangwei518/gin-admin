package elasticsearch

import (
	"context"
	"net/http"

	"github.com/wangwei518/gin-admin/internal/app/model/impl/elasticsearch/entity"
	"github.com/wangwei518/gin-admin/pkg/logger"

	"github.com/olivere/elastic/v7"
)

// Config configurations
type Config struct {
	URL      string // http://127.0.0.1:9200
	User     string // user
	Password string // password
	Index    string // index
}

// NewClient 创建elasticsearch客户端实例
func NewClient(cfg *Config) (*elastic.Client, func(), error) {
	var (
		ctx    = context.Background()
		cancel context.CancelFunc
	)

	/*
		if t := cfg.Timeout; t > 0 {
			ctx, cancel = context.WithTimeout(ctx, t)
			defer cancel()
		}

		cli, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
		if err != nil {
			return nil, nil, err
		}
	*/

	cli, err := elastic.NewClient(
		elastic.SetURL(cfg.URL),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetBasicAuth(cfg.User, cfg.Password),
	)
	if err != nil {
		return nil, nil, err
	}

	cleanFunc := func() {
		//err := cli.Disconnect(context.Background())
		err := nil
		if err != nil {
			logger.Errorf(context.Background(), "Elasticsearch impl disconnect with error: %s", err.Error())
		}
	}

	res, code, err := cli.Ping(context.Background(), nil)
	if err != nil {
		return nil, cleanFunc, err
	}
	if code != http.StatusOK {
		logger.Errorf(context.Background(), "Elasticsearch impl with error, status code=%d", code)
	}
	if res == nil {
		logger.Fatalf(context.Background(), "Elasticsearch impl with error, return result=%v", res)
	}
	return cli, cleanFunc, nil
}

// CreateIndexes 创建索引
func CreateIndexes(ctx context.Context, cli *elastic.Client) error {
	return createIndexes(
		ctx,
		cli,
		new(entity.Demo),
		new(entity.MenuAction),
		new(entity.MenuActionResource),
		new(entity.Menu),
		new(entity.RoleMenu),
		new(entity.Role),
		new(entity.UserRole),
		new(entity.User),
	)
}

type indexer interface {
	CreateIndexes(ctx context.Context, cli *elasitc.Client) error
}

func createIndexes(ctx context.Context, cli *elasitc.Client, indexes ...indexer) error {
	for _, idx := range indexes {
		err := idx.CreateIndexes(ctx, cli)
		if err != nil {
			return err
		}
	}
	return nil
}
