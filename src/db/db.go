package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/couchbase/gocb/v2"
	"github.com/couchbase/gocb/v2/search"
	"github.com/davecgh/go-spew/spew"
	"github.com/rs/xid"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
	"os"
	"time"
)

type DB interface {
	UpsertItem(ctx context.Context, inId string, item *models.Item) (id string, err error)
	GetItem(ctx context.Context, id string) (item *models.Item, err error)
	GetAllItems(ctx context.Context) (items []*models.Item, err error)
	SearchItems(ctx context.Context, index string, query string) (res []*models.SearchResult[models.Item], err error)
	DeleteItem(ctx context.Context, id string) (err error)
}

type db struct {
	cluster            *gocb.Cluster
	collectionManager  *gocb.CollectionManager
	queryIndexManager  *gocb.QueryIndexManager
	searchIndexManager *gocb.SearchIndexManager
	bucket             *gocb.Bucket
	scope              *gocb.Scope
	items              *gocb.Collection
}

func NewDB(ctx context.Context) (DB, error) {
	// Uncomment following line to enable logging
	//gocb.SetLogger(gocb.VerboseStdioLogger())

	d := &db{}
	var err error

	connectionString := os.Getenv("COUCHBASE_CONNECTION_STRING")
	bucketName := os.Getenv("COUCHBASE_BUCKET")
	username := os.Getenv("COUCHBASE_USERNAME")
	password := os.Getenv("COUCHBASE_PASSWORD")

	d.cluster, err = gocb.Connect(connectionString, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		log.Logger().Err(err)
		return nil, err
	}

	d.queryIndexManager = d.cluster.QueryIndexes()
	d.searchIndexManager = d.cluster.SearchIndexes()

	d.bucket = d.cluster.Bucket(bucketName)

	err = d.bucket.WaitUntilReady(5*time.Second, &gocb.WaitUntilReadyOptions{
		Context: ctx,
	})
	if err != nil {
		log.Logger().Err(err)
		return nil, err
	}

	d.collectionManager = d.bucket.Collections()

	err = d.collectionManager.CreateScope("0",
		&gocb.CreateScopeOptions{Context: ctx})
	if err != nil {
		if !errors.Is(err, gocb.ErrScopeExists) {
			log.Logger().Err(err)
			return nil, err
		}
	}
	d.scope = d.bucket.Scope("0")
	err = d.collectionManager.CreateCollection(gocb.CollectionSpec{
		Name:      "items",
		ScopeName: "0",
	}, &gocb.CreateCollectionOptions{
		Context: ctx,
	})
	if err != nil {
		if !errors.Is(err, gocb.ErrCollectionExists) {
			log.Logger().Err(err)
			return nil, err
		}
	}
	d.items = d.scope.Collection("items")

	if err = d.queryIndexManager.CreatePrimaryIndex(bucketName, nil); err != nil {
		if !errors.Is(err, gocb.ErrIndexExists) {
			log.Logger().Err(err)
			return nil, err
		}
	}

	for _, fieldName := range []string{"created", "updated", "title", "amount", "unit", "bought"} {
		if err := d.queryIndexManager.CreateIndex(d.bucket.Name(), "ix_"+fieldName, []string{fieldName},
			&gocb.CreateQueryIndexOptions{Context: ctx}); err != nil {
			if !errors.Is(err, gocb.ErrIndexExists) {
				log.Logger().Err(err)
				return nil, err
			}
		}
	}

	//if err = d.searchIndexManager.UpsertIndex(gocb.SearchIndex{
	//	UUID:         "title-index",
	//	Name:         "title-index",
	//	SourceName:   d.bucket.Name(),
	//	Type:         "fulltext-index",
	//	Params:       nil,
	//	SourceUUID:   "",
	//	SourceParams: nil,
	//	SourceType:   "couchbase",
	//	PlanParams:   nil,
	//}, &gocb.UpsertSearchIndexOptions{Context: ctx}); err != nil {
	//	if !errors.Is(err, gocb.ErrIndexExists) {
	//		log.Println(err)
	//		return nil, err
	//	}
	//}

	return d, nil
}

func (d *db) UpsertItem(ctx context.Context, inId string, item *models.Item) (outId string, err error) {
	outId = inId
	if outId == "" {
		outId = xid.New().String()
	}
	item.Base = models.Base{
		Created: time.Now().UTC().UnixMilli(),
		Updated: time.Now().UTC().UnixMilli(),
	}

	_, err = d.items.Upsert(outId, item,
		&gocb.UpsertOptions{Context: ctx})
	if err != nil {
		log.Logger().Err(err)
		return

	}
	log.Logger().Info().Msgf("Item created: %s\n", inId)
	return
}

func Key(prefix string, id string) string {
	return fmt.Sprintf("%s:%s", prefix, id)
}
func (d *db) GetItem(ctx context.Context, id string) (item *models.Item, err error) {
	getResult, err := d.items.Get(id,
		&gocb.GetOptions{Context: ctx})
	if err != nil {
		log.Logger().Err(err)
		return

	}

	item = &models.Item{}
	err = getResult.Content(item)
	if err != nil {
		log.Logger().Err(err)
		return

	}
	spew.Dump(item)

	return
}

func (d *db) GetAllItems(ctx context.Context) (items []*models.Item, err error) {
	queryResult, err := d.scope.Query(
		fmt.Sprintf("SELECT x.* FROM items x"),
		&gocb.QueryOptions{Adhoc: true, Context: ctx},
	)
	if err != nil {
		log.Logger().Err(err)
		return
	}

	// Print each found Row
	for queryResult.Next() {
		var item models.Item
		err = queryResult.Row(&item)
		if err != nil {
			log.Logger().Err(err)
			return
		}
		items = append(items, &item)
	}

	if err = queryResult.Err(); err != nil {
		log.Logger().Err(err)
		return
	}

	spew.Dump(items)
	return
}

func (d *db) SearchItems(ctx context.Context, index string, query string) (itemSearchResults []*models.SearchResult[models.Item], err error) {
	matchResult, err := d.cluster.SearchQuery(
		index,
		search.NewConjunctionQuery(
			search.NewMatchQuery(query),
			//search.NewDateRangeQuery().Start("2019-01-01", true).End("2029-02-01", false),
		),
		&gocb.SearchOptions{
			Limit:   100,
			Fields:  []string{"title"},
			Context: ctx,
		},
	)
	if err != nil {
		panic(err)
	}

	// Print each found Row
	for matchResult.Next() {
		var itemSearchResult models.SearchResult[models.Item]
		row := matchResult.Row()
		itemSearchResult.ID = row.ID
		itemSearchResult.Score = row.Score
		err = row.Fields(&itemSearchResult.Data)
		if err != nil {
			log.Logger().Err(err)
			return
		}
		itemSearchResults = append(itemSearchResults, &itemSearchResult)
	}

	if err = matchResult.Err(); err != nil {
		log.Logger().Err(err)
		return
	}

	spew.Dump(itemSearchResults)
	return

}

func (d *db) DeleteItem(ctx context.Context, id string) (err error) {
	_, err = d.items.Remove(id,
		&gocb.RemoveOptions{Context: ctx})
	if err != nil {
		log.Logger().Err(err)
		return
	}
	log.Logger().Info().Msgf("Item deleted: %s\n", id)
	return
}
