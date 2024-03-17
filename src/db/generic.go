package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/log"
)

func init() {
	// Uncomment following line to enable logging
	//gocb.SetLogger(gocb.DefaultStdioLogger())
}

type GenericDB interface {
	Ping(ctx context.Context) (report string, err error)
}

type db struct {
	label              string
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

	config *config.Config
}

func NewGenericDB(ctx context.Context, label string, cfg *config.Config) (GenericDB, error) {
	db := &db{
		fields: []string{"title", "amount", "unit", "bought", "shop"},
		bought: sql.NullBool{
			Bool:  false,
			Valid: false,
		},
		config: cfg,
		label:  label,
	}
	err := db.init(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *db) init(ctx context.Context) error {

	var err error

	connectionString := d.config.CouchbaseConnectionString
	bucketName := d.config.CouchbaseBucketName
	username := d.config.CouchbaseUsername
	password := d.config.CouchbasePassword

	d.cluster, err = gocb.Connect(connectionString, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		log.Logger(ctx).Err(err).Msg("gocb.Connect")
		return err
	}

	d.searchIndexManager = d.cluster.SearchIndexes()
	if err != nil {
		log.Logger(ctx).Err(err).Msg("d.cluster.SearchIndexes")
		return err
	}

	tmpBucket := d.cluster.Bucket(bucketName)
	err = tmpBucket.WaitUntilReady(5*time.Second, &gocb.WaitUntilReadyOptions{
		Context: ctx,
	})
	if err != nil {
		RAMQuotaMB, err := strconv.Atoi(d.config.CouchbaseRamQuotaMB)
		if err != nil {
			log.Logger(ctx).Err(err).Msg("RAMQuotaMB")
			return err
		}
		err = d.cluster.Buckets().CreateBucket(gocb.CreateBucketSettings{
			BucketSettings: gocb.BucketSettings{
				Name:         bucketName,
				RAMQuotaMB:   uint64(RAMQuotaMB),
				FlushEnabled: true,
				BucketType:   gocb.CouchbaseBucketType,
			},
		}, &gocb.CreateBucketOptions{
			Context: ctx,
		})
		if err != nil && !errors.Is(err, gocb.ErrBucketExists) {
			log.Logger(ctx).Err(err).Msg("cluster.Buckets().CreateBucket")
			return err
		}
	}

	d.bucket = d.cluster.Bucket(bucketName)

	err = d.bucket.WaitUntilReady(5*time.Second, &gocb.WaitUntilReadyOptions{
		Context: ctx,
	})
	if err != nil {
		log.Logger(ctx).Err(err).Msg("bucket.WaitUntilReady")
		return err
	}

	d.collectionManager = d.bucket.Collections()

	err = d.collectionManager.CreateScope("0",
		&gocb.CreateScopeOptions{Context: ctx})
	if err != nil {
		if !errors.Is(err, gocb.ErrScopeExists) {
			log.Logger(ctx).Err(err).Msg("collectionManager.CreateScope")
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
				log.Logger(ctx).Err(err).Msg("collectionManager.CreateCollection")
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
				log.Logger(ctx).Err(err).Msg("indexManager.CreatePrimaryIndex")
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
					log.Logger(ctx).Err(err).Msg("indexManager.CreateIndex")
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

func (d *db) start(ctx context.Context, name string) (context.Context, trace.Span, zerolog.Logger, func()) {
	ctx, span := d.config.Tracer.Start(ctx, name)
	span.SetAttributes(attribute.String("handlerLabel", d.label))
	l := log.Logger(ctx).With().Any("handlerLabel", d.label).Logger()

	f := func() {
		span.End()
	}
	return ctx, span, l, f
}
