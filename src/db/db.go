package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/couchbase/gocb/v2"
	"github.com/couchbase/gocb/v2/search"
	"github.com/davecgh/go-spew/spew"
	"github.com/rs/xid"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
	"os"
	"slices"
	"sync"
	"time"
)

type DB interface {
	UpsertItem(ctx context.Context, inId string, item *models.Item) (id string, err error)
	GetItem(ctx context.Context, id string) (item *models.Item, err error)
	GetItems(ctx context.Context, q *PaginationQuery) (items []*models.ItemWithID, total int, err error)
	SearchItems(ctx context.Context, query string) (items []*models.ItemWithID, err error)
	Ping(ctx context.Context) (report string, err error)
}

type db struct {
	cluster            *gocb.Cluster
	collectionManager  *gocb.CollectionManager
	searchIndexManager *gocb.SearchIndexManager
	bucket             *gocb.Bucket
	scope              *gocb.Scope
	collection         *gocb.Collection
	indexManager       *gocb.CollectionQueryIndexManager
	fields             []string
}

var items *db
var muItems sync.Mutex

func NewItemsDB(ctx context.Context) (DB, error) {
	muItems.Lock()
	defer muItems.Unlock()

	if items != nil {
		return items, nil
	}
	// Uncomment following line to enable logging
	//gocb.SetLogger(gocb.VerboseStdioLogger())

	items = &db{
		fields: []string{"title", "amount", "unit", "bought", "shop"},
	}
	var err error

	connectionString := os.Getenv("COUCHBASE_CONNECTION_STRING")
	bucketName := os.Getenv("COUCHBASE_BUCKET")
	username := os.Getenv("COUCHBASE_USERNAME")
	password := os.Getenv("COUCHBASE_PASSWORD")

	items.cluster, err = gocb.Connect(connectionString, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		log.Logger().Err(err)
		return nil, err
	}

	items.searchIndexManager = items.cluster.SearchIndexes()

	items.bucket = items.cluster.Bucket(bucketName)

	err = items.bucket.WaitUntilReady(5*time.Second, &gocb.WaitUntilReadyOptions{
		Context: ctx,
	})
	if err != nil {
		log.Logger().Err(err)
		return nil, err
	}

	items.collectionManager = items.bucket.Collections()

	err = items.collectionManager.CreateScope("0",
		&gocb.CreateScopeOptions{Context: ctx})
	if err != nil {
		if !errors.Is(err, gocb.ErrScopeExists) {
			log.Logger().Err(err)
			return nil, err
		}
	}
	items.scope = items.bucket.Scope("0")
	err = items.collectionManager.CreateCollection(gocb.CollectionSpec{
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
	items.collection = items.scope.Collection("items")

	items.indexManager = items.collection.QueryIndexes()

	if err = items.indexManager.CreatePrimaryIndex(&gocb.CreatePrimaryQueryIndexOptions{
		IgnoreIfExists: false,
		Deferred:       false,
		Context:        ctx,
	}); err != nil {
		if !errors.Is(err, gocb.ErrIndexExists) {
			log.Logger().Err(err)
			return nil, err
		}
	}

	for _, fieldName := range items.fields {
		if err := items.indexManager.CreateIndex("ix_"+fieldName, []string{fieldName},
			&gocb.CreateQueryIndexOptions{
				IgnoreIfExists: false,
				Deferred:       false,
				Context:        ctx,
			}); err != nil {
			if !errors.Is(err, gocb.ErrIndexExists) {
				log.Logger().Err(err)
				return nil, err
			}
		}
	}

	//if err = instance.searchIndexManager.UpsertIndex(gocb.SearchIndex{
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

	return items, nil
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

	_, err = d.collection.Upsert(outId, item,
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
	getResult, err := d.collection.Get(id,
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

type PaginationQuery struct {
	Start int
	End   int
	Sort  string
	Order string
	Query string
}

func (d *db) GetItems(ctx context.Context, q *PaginationQuery) (items []*models.ItemWithID, total int, err error) {
	query := "SELECT meta(x).id, x.* FROM items x"
	queryTotal := "SELECT COUNT(*) as total FROM items x"
	if q.Order == "" {
		q.Order = "ASC"
	}
	if q.Sort != "" {
		query += fmt.Sprintf("\nORDER BY x.%s %s, meta(x).id ASC", q.Sort, q.Order)
	} else {
		query += fmt.Sprintf("\nORDER BY meta(x).id ASC")
	}
	if q.Start != 0 {
		query += fmt.Sprintf("\nOFFSET %d ", q.Start)
	}
	if q.End != 0 {
		query += fmt.Sprintf("\nLIMIT %d ", q.End-q.Start)
	}

	log.Logger().Info().Msgf("Query: %s", query)
	params := make(map[string]interface{})
	queryResult, err := d.scope.Query(query, &gocb.QueryOptions{Adhoc: true, Context: ctx, NamedParameters: params})
	if err != nil {
		log.Logger().Err(err)
		return
	}
	items = []*models.ItemWithID{}
	for queryResult.Next() {
		var item models.ItemWithID
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

	paramsTotal := make(map[string]interface{})
	queryResultTotal, err := d.scope.Query(queryTotal, &gocb.QueryOptions{Adhoc: true, Context: ctx, NamedParameters: paramsTotal})
	if err != nil {
		log.Logger().Err(err)
		return
	}
	var totalResult models.Total
	err = queryResultTotal.One(&totalResult)
	if err != nil {
		log.Logger().Err(err)
		return
	}
	total = totalResult.Total

	return
}

func (d *db) SearchItems(ctx context.Context, query string) (items []*models.ItemWithID, err error) {
	matchResult, err := d.cluster.SearchQuery(
		"title-index",
		search.NewConjunctionQuery(
			search.NewMatchQuery(query),
			//search.NewDateRangeQuery().Start("2019-01-01", true).End("2029-02-01", false),
		),
		&gocb.SearchOptions{
			Limit:   1000,
			Fields:  d.fields,
			Context: ctx,
		},
	)
	if err != nil {
		return nil, err

	}

	itemSearchResults := make([]*models.ItemSearchResult, 0)
	// Print each found Row
	for matchResult.Next() {
		var itemSearchResult models.ItemSearchResult
		row := matchResult.Row()
		err = row.Fields(&itemSearchResult)
		if err != nil {
			log.Logger().Err(err)
			return
		}
		itemSearchResult.ID = row.ID
		itemSearchResult.Score = row.Score
		itemSearchResults = append(itemSearchResults, &itemSearchResult)
	}
	if err = matchResult.Err(); err != nil {
		log.Logger().Err(err)
		return
	}

	slices.SortFunc(itemSearchResults, func(i, j *models.ItemSearchResult) int {
		return int(j.Score - i.Score)
	})

	items = make([]*models.ItemWithID, 0, len(itemSearchResults))
	for _, result := range itemSearchResults {
		items = append(items, &models.ItemWithID{
			Item: models.Item{
				Title:  result.Title,
				Amount: result.Amount,
				Unit:   result.Unit,
				Bought: result.Bought,
				Shop:   result.Shop,
			},
			ID: result.ID,
		})
	}

	return

}

func (d *db) DeleteItem(ctx context.Context, id string) (err error) {
	_, err = d.collection.Remove(id,
		&gocb.RemoveOptions{Context: ctx})
	if err != nil {
		log.Logger().Err(err)
		return
	}
	log.Logger().Info().Msgf("Item deleted: %s\n", id)
	return
}

func (d *db) Ping(ctx context.Context) (report string, err error) {
	pings, err := d.bucket.Ping(&gocb.PingOptions{
		ReportID:     "ping",
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue},
		Context:      ctx,
	})
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(pings)
	if err != nil {
		return "", err
	}

	for service, pingReports := range pings.Services {
		if service != gocb.ServiceTypeKeyValue {
			err = fmt.Errorf("we got a service type that we didn't ask for")
			return "", err
		}

		for _, pingReport := range pingReports {
			if pingReport.State != gocb.PingStateOk {
				err = fmt.Errorf("we got a service state that is not OK")
			}
		}
	}

	return string(b), err
}

func InitDB(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	itemsDB, err := NewItemsDB(ctx)
	if err != nil {
		log.Logger().Error().Err(err)
		return
	}
	items := []*models.Item{
		{
			Title:  "Cottage Cheese",
			Amount: 1,
			Unit:   "pc",
			Bought: false,
			Shop:   "Rewe",
		},
		{
			Title:  "Avocado",
			Amount: 2,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Banana",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Milk",
			Amount: 2,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Bread",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Sosages",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Meat",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Creme Fraiche",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Wine",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Napkins",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Tomatoes",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Cucumber",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Ananas",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Plums",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
		{
			Title:  "Clementines",
			Amount: 1,
			Unit:   "pc",
			Bought: true,
			Shop:   "Edeka",
		},
	}

	for _, item := range items {
		_, err = itemsDB.UpsertItem(ctx, Key("item", item.Title), item)
		if err != nil {
			log.Logger().Error().Err(err)
			return err
		}
	}

	return
}
