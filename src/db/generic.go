package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/couchbase/gocb/v2"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
	"os"
	"time"
)

type GenericDB interface {
	Ping(ctx context.Context) (report string, err error)
}

type db struct {
	cluster            *gocb.Cluster
	collectionManager  *gocb.CollectionManager
	searchIndexManager *gocb.SearchIndexManager
	bucket             *gocb.Bucket
	scope              *gocb.Scope
	collectionName     string
	collection         *gocb.Collection
	indexManager       *gocb.CollectionQueryIndexManager
	fields             []string

	bought sql.NullBool
}

func NewGenericDB(ctx context.Context) (GenericDB, error) {
	db := &db{
		fields: []string{"title", "amount", "unit", "bought", "shop"},
		bought: sql.NullBool{
			Bool:  false,
			Valid: false,
		},
	}
	err := db.init(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *db) init(ctx context.Context) error {

	// Uncomment following line to enable logging
	//gocb.SetLogger(gocb.VerboseStdioLogger())

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
		return err
	}

	d.searchIndexManager = d.cluster.SearchIndexes()

	d.bucket = d.cluster.Bucket(bucketName)

	err = d.bucket.WaitUntilReady(5*time.Second, &gocb.WaitUntilReadyOptions{
		Context: ctx,
	})
	if err != nil {
		log.Logger().Err(err)
		return err
	}

	d.collectionManager = d.bucket.Collections()

	err = d.collectionManager.CreateScope("0",
		&gocb.CreateScopeOptions{Context: ctx})
	if err != nil {
		if !errors.Is(err, gocb.ErrScopeExists) {
			log.Logger().Err(err)
			return err
		}
	}
	d.scope = d.bucket.Scope("0")
	if d.collectionName != "" {
		err = d.collectionManager.CreateCollection(gocb.CollectionSpec{
			Name:      d.collectionName,
			ScopeName: "0",
		}, &gocb.CreateCollectionOptions{
			Context: ctx,
		})
		if err != nil {
			if !errors.Is(err, gocb.ErrCollectionExists) {
				log.Logger().Err(err)
				return err
			}
		}
		d.collection = d.scope.Collection(d.collectionName)

		d.indexManager = d.collection.QueryIndexes()

		if err = d.indexManager.CreatePrimaryIndex(&gocb.CreatePrimaryQueryIndexOptions{
			IgnoreIfExists: false,
			Deferred:       false,
			Context:        ctx,
		}); err != nil {
			if !errors.Is(err, gocb.ErrIndexExists) {
				log.Logger().Err(err)
				return err
			}
		}

		for _, fieldName := range d.fields {
			if err := d.indexManager.CreateIndex("ix_"+fieldName, []string{fieldName},
				&gocb.CreateQueryIndexOptions{
					IgnoreIfExists: false,
					Deferred:       false,
					Context:        ctx,
				}); err != nil {
				if !errors.Is(err, gocb.ErrIndexExists) {
					log.Logger().Err(err)
					return err
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
	}

	return nil
}

func Key(prefix string, id string) string {
	return fmt.Sprintf("%s:%s", prefix, id)
}

type PaginationQuery struct {
	Start int
	End   int
	Sort  string
	Order string
	Query string
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

	itemsDB, err := NewItemsDB(ctx, sql.NullBool{
		Bool:  true,
		Valid: true,
	})
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
