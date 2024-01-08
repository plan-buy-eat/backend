package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/couchbase/gocb/v2"
	"github.com/couchbase/gocb/v2/search"
	"github.com/rs/xid"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
	"slices"
	"time"
)

type ItemsDB interface {
	UpsertItem(ctx context.Context, inId string, item *models.Item) (id string, err error)
	GetItem(ctx context.Context, id string) (item *models.Item, err error)
	GetItems(ctx context.Context, q *PaginationQuery) (items []*models.ItemWithID, total int, err error)
	SearchItems(ctx context.Context, query string) (items []*models.ItemWithID, err error)
	DeleteItem(ctx context.Context, id string) (err error)
	BuyItem(ctx context.Context, id string, bought bool) (err error)
}

func NewItemsDB(ctx context.Context, bought sql.NullBool) (ItemsDB, error) {
	db := &db{
		collectionName: "items",
		fields:         []string{"title", "amount", "unit", "bought", "shop"},
		bought:         bought,
	}
	err := db.init(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (d *db) UpsertItem(ctx context.Context, inId string, item *models.Item) (outId string, err error) {
	outId = inId
	if outId == "" {
		outId = xid.New().String()
	}
	if item == nil {
		item = &models.Item{Base: models.Base{}}
	}
	item.Base.Updated = time.Now().UTC().UnixMilli()
	if item.Base.Created == 0 {
		item.Base.Created = time.Now().UTC().UnixMilli()
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

	if d.bought.Valid {
		if item.Bought != d.bought.Bool {
			return nil, nil
		}
	}

	return
}

func (d *db) BuyItem(ctx context.Context, id string, bought bool) (err error) {

	mops := []gocb.MutateInSpec{
		gocb.ReplaceSpec("bought", bought, &gocb.ReplaceSpecOptions{}),
	}
	if d.collection == nil {
		err = fmt.Errorf("collection is nil")
		log.Logger().Err(err)
		return
	}
	_, err = d.collection.MutateIn(id, mops, &gocb.MutateInOptions{
		Context: ctx,
		//Timeout: 10050 * time.Millisecond,
	})
	if err != nil {
		log.Logger().Err(err)
		return
	}

	return
}

func (d *db) GetItems(ctx context.Context, q *PaginationQuery) (items []*models.ItemWithID, total int, err error) {
	query := "SELECT meta(x).id, x.* FROM items x"
	queryTotal := "SELECT COUNT(*) as total FROM items x"

	if d.bought.Valid {
		if d.bought.Bool {
			query += fmt.Sprintf("\nWHERE x.bought = true")
			queryTotal += fmt.Sprintf("\nWHERE x.bought = true")
		} else {
			query += fmt.Sprintf("\nWHERE x.bought = false")
			queryTotal += fmt.Sprintf("\nWHERE x.bought = false")
		}
	}

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
			Limit:   10000,
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
