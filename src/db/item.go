package db

import (
	"context"
	"database/sql"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/xid"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
)

type ItemsDB interface {
	UpsertItem(ctx context.Context, item *models.ItemWithID) (err error)
	GetItem(ctx context.Context, id string) (item *models.ItemWithID, err error)
	GetItems(ctx context.Context, q *PaginationQuery, searchQuery string, boughtLast bool) (items []*models.ItemWithID, total int, err error)
	DeleteItem(ctx context.Context, id string) (err error)
	BuyItem(ctx context.Context, id string, bought *bool) (err error)
	InitDB(ctx context.Context) (err error)
}

func NewItemsDB(ctx context.Context, cfg *config.Config, bought sql.NullBool) (ItemsDB, error) {
	label := "items"
	if bought.Valid {
		if bought.Bool {
			label = "bought"
		} else {
			label = "toBuy"
		}
	}
	db := &db{
		collectionName: "items",
		fields:         []string{"title", "amount", "unit", "bought", "shop"},
		bought:         bought,
		config:         cfg,
		label:          label,
	}
	err := db.init(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *db) UpsertItem(ctx context.Context, item *models.ItemWithID) (err error) {
	ctx, _, _, def := d.start(ctx, log.GetFuncName())
	defer def()

	outId := item.ID
	if outId == "" {
		outId = xid.New().String()
	}
	if item == nil {
		item = &models.ItemWithID{Item: models.Item{Base: models.Base{}}}
	}
	item.Base.Updated = time.Now().UTC().UnixMilli()
	if item.Base.Created == 0 {
		item.Base.Created = time.Now().UTC().UnixMilli()
	}

	_, err = d.collection.Upsert(outId, item.Item,
		&gocb.UpsertOptions{Context: ctx})
	if err != nil {
		log.Logger(ctx).Err(err)
		return
	}
	log.Logger(ctx).Info().Msgf("Item created: %s\n", item.ID)
	var item2 *models.ItemWithID
	item2, err = d.GetItem(ctx, outId)
	if err != nil {
		log.Logger(ctx).Err(err)
		return
	}
	*item = *item2
	return
}

func (d *db) GetItem(ctx context.Context, id string) (item *models.ItemWithID, err error) {
	ctx, _, _, def := d.start(ctx, log.GetFuncName())
	defer def()

	getResult, err := d.collection.Get(id,
		&gocb.GetOptions{Context: ctx})
	if err != nil {
		log.Logger(ctx).Err(err)
		return

	}

	item = &models.ItemWithID{}
	err = getResult.Content(item)
	if err != nil {
		log.Logger(ctx).Err(err)
		return
	}

	if d.bought.Valid {
		if item.Bought != d.bought.Bool {
			return nil, nil
		}
	}
	item.ID = id

	return
}

func (d *db) BuyItem(ctx context.Context, id string, bought *bool) (err error) {
	ctx, _, _, def := d.start(ctx, log.GetFuncName())
	defer def()

	var b bool
	if bought != nil {
		b = *bought
	} else {
		var item *models.ItemWithID
		item, err = d.GetItem(ctx, id)
		if err != nil {
			log.Logger(ctx).Err(err).Msg("MutateIn")
			return
		}
		b = !item.Bought
	}

	mops := []gocb.MutateInSpec{
		gocb.ReplaceSpec("bought", b, &gocb.ReplaceSpecOptions{}),
	}
	if d.collection == nil {
		err = fmt.Errorf("collection is nil")
		log.Logger(ctx).Err(err)
		return
	}
	_, err = d.collection.MutateIn(id, mops, &gocb.MutateInOptions{
		Context: ctx,
		//Timeout: 10050 * time.Millisecond,
	})
	if err != nil {
		log.Logger(ctx).Err(err).Msg("MutateIn")
		return
	}

	return
}

func (d *db) GetItems(ctx context.Context, q *PaginationQuery, searchQuery string, boughtLast bool) (items []*models.ItemWithID, total int, err error) {
	ctx, _, _, def := d.start(ctx, log.GetFuncName())
	defer def()

	searchQuery = strings.TrimSpace(searchQuery)

	query := "SELECT meta(x).id, x.* FROM items x WHERE 1=1"
	queryTotal := "SELECT COUNT(*) as total FROM items x WHERE 1=1"

	if d.bought.Valid {
		if d.bought.Bool {
			query += fmt.Sprintf("\nAND x.bought = true")
			queryTotal += fmt.Sprintf("\nAND x.bought = true")
		} else {
			query += fmt.Sprintf("\nAND x.bought = false")
			queryTotal += fmt.Sprintf("\nAND x.bought = false")
		}
	}

	if searchQuery != "" {
		query += fmt.Sprintf("\nAND SEARCH(x, $searchQuery)")
		queryTotal += fmt.Sprintf("\nAND SEARCH(x, $searchQuery)")
	}

	if q.Order == "" {
		q.Order = "ASC"
	}

	order := ""

	if boughtLast {
		//if order != "" {
		//	order += ", "
		//}
		order += fmt.Sprintf("x.bought asc")
	}
	if q.Sort != "" {
		if order != "" {
			order += ", "
		}
		order += fmt.Sprintf("x.%s %s", q.Sort, q.Order)
	}

	if order != "" {
		query += fmt.Sprintf("\nORDER BY " + order)
	}

	if q.Start != 0 {
		query += fmt.Sprintf("\nOFFSET %d ", q.Start)
	}
	if q.End != 0 {
		query += fmt.Sprintf("\nLIMIT %d ", q.End-q.Start)
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() (err error) {
		ctx, span := d.config.Tracer.Start(ctx, "main")
		defer span.End()

		l := log.Logger(ctx).With().Any("query", "main").Logger()
		l.Info().Msgf("Main query: %s", query)
		params := map[string]interface{}{
			"searchQuery": searchQuery,
		}
		queryResult, err := d.scope.Query(query, &gocb.QueryOptions{Adhoc: true, Context: ctx, NamedParameters: params})
		if err != nil {
			l.Err(err).Msg("Query")
			return
		}
		items = []*models.ItemWithID{}
		for queryResult.Next() {
			var item models.ItemWithID
			err = queryResult.Row(&item)
			if err != nil {
				l.Err(err).Msg("Row")
				return
			}
			items = append(items, &item)
		}
		if err = queryResult.Err(); err != nil {
			l.Err(err).Msg("queryResult.Err")
			return
		}
		return
	})

	eg.Go(func() (err error) {
		ctx, span := d.config.Tracer.Start(ctx, "total")
		defer span.End()
		l := log.Logger(ctx).With().Any("query", "total").Logger()

		l.Info().Msgf("Total query: %s", queryTotal)
		paramsTotal := map[string]interface{}{
			"searchQuery": searchQuery,
		}
		queryResultTotal, err := d.scope.Query(queryTotal, &gocb.QueryOptions{Adhoc: true, Context: ctx, NamedParameters: paramsTotal})
		if err != nil {
			l.Err(err).Msg("Query")
			return
		}
		var totalResult models.Total
		err = queryResultTotal.One(&totalResult)
		if err != nil {
			l.Err(err).Msg("queryResultTotal.One")
			return
		}
		total = totalResult.Total
		return
	})

	if err = eg.Wait(); err != nil {
		return
	}

	return
}

func (d *db) DeleteItem(ctx context.Context, id string) (err error) {
	ctx, _, _, def := d.start(ctx, log.GetFuncName())
	defer def()

	_, err = d.collection.Remove(id,
		&gocb.RemoveOptions{Context: ctx})
	if err != nil {
		log.Logger(ctx).Err(err)
		return
	}
	log.Logger(ctx).Info().Msgf("Item deleted: %s\n", id)
	return
}
