package elasticsearch

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/olivere/elastic/v7"
)

// Config configurations
type Config struct {
	URL      string // http://127.0.0.1:9200
	User     string // user
	Password string // password
	Index    string // index
}

// BlackListDoc ...
type BlackListDoc struct {
	Token   string    `json:"token"`
	Issuer  string    `json:"issuer,omitempty"`
	Created time.Time `json:"created,omitempty"`
}

// NewStore build elasticsearch connection
func NewStore(cfg *Config) (*Store, error) {

	cli, err := elastic.NewClient(
		elastic.SetURL(cfg.URL),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetBasicAuth(cfg.User, cfg.Password),
	)
	if err != nil {
		return nil, err
	}

	/*
		// Use the IndexExists service to check if a specified index exists.
		indexBlackList := cfg.IndexPrefix+"jwt_blacklist"
		exists, err := s.cli.IndexExists(indexBlackList).Do(ctx.Background())
		if err != nil {
			// Handle error
			panic(err)
		}

		if !exists {
			fmt.Printf("[Elasticsearch] index '%s' does not exist.", indexBlackList)
			// Create a new index.
			mapping := `
				{
					"settings":{
						"number_of_shards":1,
						"number_of_replicas":0
					},
					"mappings":{
						"doc":{
							"properties":{
								"token":{
									"type":"keyword"
								},
								"issuer":{
									"type":"text",
								},
								"created":{
									"type":"date"
								}
							}
						}
					}
				}
				`
			createIndex, err := client.CreateIndex(indexBlackList).Body(mapping).Do(context.Background())
			if err != nil {
				// Handle error
				panic(err)
			}
			if !createIndex.Acknowledged {
				// Not acknowledged
				fmt.Printf("[Elasticsearch] create index '%s' failed, no acknowledge.\n", indexBlackList)
				return nil
			}

			fmt.Printf("[Elasticsearch] created index '%s' successfuly.\n", indexBlackList)

		} else {
			fmt.Printf("[Elasticsearch] index '%s' exists.\n", indexBlackList)
		}
	*/

	return &Store{
		Cli:   cli,
		Index: cfg.Index,
	}, nil
}

// Store client handler
type Store struct {
	Cli   *elastic.Client
	Index string
}

// Set ...
func (s *Store) Set(ctx context.Context, tokenString string, expiration time.Duration) error {

	_ = expiration

	// string is also available, for example
	// doc := `{"user" : "olivere", "message" : "It's a Raggy Waltz"}`
	// s.cli.Index().....BodyString(doc)
	doc := BlackListDoc{
		Token:   tokenString,
		Issuer:  "FIXME",
		Created: time.Now(), //time.Now().UTC(),
	}
	//		Id().
	// omit Id to generate _id dynamicly
	put, err := s.Cli.Index().
		Index(s.Index).
		BodyJson(doc).
		Do(ctx)
	if err != nil {
		// Handle error
		return err
	}
	fmt.Printf("[Elasticsearch] Create index='%s', id='%s', token=%s, issuer=%s, created=%s\n", put.Index, put.Id, doc.Token, doc.Issuer, doc.Created)

	// Refresh to make sure the documents are searchable.
	_, err = s.Cli.Refresh().Index(s.Index).Do(ctx)
	fmt.Printf("[Elasticsearch] Refresh index='%s', id='%s', token=%s, issuer=%s, created=%s\n", put.Index, put.Id, doc.Token, doc.Issuer, doc.Created)
	if err != nil {
		return err
	}

	return nil
}

// Delete ...
func (s *Store) Delete(ctx context.Context, tokenString string) error {

	// DeleteByQuery
	matchQuery := elastic.NewMatchQuery("token", tokenString)
	res, err := s.Cli.DeleteByQuery().
		Index(s.Index).    // search in index "ga_jwt_blacklist"
		Query(matchQuery). // specify the query
		Pretty(true).      // pretty print request and response JSON
		Do(ctx)            // execute

	fmt.Printf("[Elasticsearch] DeleteByQuery index=%s, token=%s \n", s.Index, tokenString)

	if err != nil {
		// Handle error
		return err
	}

	if res == nil {
		err := fmt.Errorf("[Elasticsearch] DeleteByQuery, index=%s, token=%s, res=%v", s.Index, tokenString, res)
		return err
	}

	_, err = s.Cli.Refresh().Index(s.Index).Do(ctx)
	if err != nil {
		fmt.Printf("[Elasticsearch] DeleteByQuery index=%s, token=%s, refresh error\n", s.Index, tokenString)
		return err
	}

	return nil

}

// Check ...
//From(0).Size(10).        	// take 10 documents
func (s *Store) Check(ctx context.Context, tokenString string) (bool, error) {

	matchQuery := elastic.NewMatchQuery("token", tokenString)
	//matchQuery := elastic.NewMatchQuery("token", tokenString)
	//matchQuery := elastic.NewMatchAllQuery()
	searchResult, err := s.Cli.Search().
		Index(s.Index).        // search in index "ga_jwt_blacklist"
		Query(matchQuery).     // specify the query
		Sort("created", true). // sort by "created" field, ascending
		Pretty(true).          // pretty print request and response JSON
		From(0).Size(10).      // take documents 0-9
		Do(ctx)                // execute

	fmt.Printf("[Elasticsearch] Query index='%s', token='%s', hits=%d, runtime=%d milliseconds, \n", s.Index, tokenString, searchResult.TotalHits(), searchResult.TookInMillis)

	if err != nil {
		// Handle error
		return false, err
	}

	if searchResult.TotalHits() != 1 {

		var ttyp BlackListDoc
		for _, item := range searchResult.Each(reflect.TypeOf(ttyp)) {
			t := item.(BlackListDoc)
			fmt.Printf("Document, token=%s, issuer=%s, created=%s\n", t.Token, t.Issuer, t.Created)
		}

		return false, nil
	}

	return true, nil

}

// Close ...
func (s *Store) Close() error {
	// seems there is no close function in olivere/elasticsearch
	// use flush instead
	//return s.Cli.Flush()
	return nil
}
