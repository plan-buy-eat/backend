package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/db"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"net/http"
	"strconv"
	"time"
)

type ItemHandler interface {
	GetItems(c *gin.Context)
	GetItem(c *gin.Context)
	BuyItem(c *gin.Context)
	RestoreItem(c *gin.Context)
	ToggleItem(c *gin.Context)
	DeleteItem(c *gin.Context)
	CreateItem(c *gin.Context)
	EditItem(c *gin.Context)
}

type itemHandler struct {
	genericHandler
	bought     sql.NullBool
	boughtLast bool
	apiCalls   metric.Int64Histogram
}

func NewItemHandler(ctx context.Context, label string, bought sql.NullBool, boughtLast bool) (ItemHandler, error) {
	h := &itemHandler{
		genericHandler: genericHandler{
			config: config.Get(ctx),
			label:  label,
		},
		bought:     bought,
		boughtLast: boughtLast,
	}

	var err error
	h.apiCalls, err = h.config.Meter.Int64Histogram(
		"api.calls",
		metric.WithDescription("API calls histogram"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *itemHandler) GetItem(c *gin.Context) {
	ctx, _, _, def := h.start(c, log.GetFuncName())
	defer def()
	log.Logger(ctx).Info().Msg("Start")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	itemOut, err := itemsDB.GetItem(ctx, id)
	if err != nil {
		h.err(c, "getting an item", err)
		return
	}

	h.res(c, itemOut)
}

func (h *itemHandler) BuyItem(c *gin.Context) {
	ctx, _, _, def := h.start(c, log.GetFuncName())
	defer def()
	log.Logger(ctx).Info().Msg("Start")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	b := true
	err = itemsDB.BuyItem(ctx, id, &b)
	if err != nil {
		h.err(c, "buying an item", err)
		return
	}
	h.resWithStatus(c, http.StatusOK, models.ID{ID: id})
}

func (h *itemHandler) RestoreItem(c *gin.Context) {
	ctx, _, _, def := h.start(c, log.GetFuncName())
	defer def()
	log.Logger(ctx).Info().Msg("Start")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	b := false
	err = itemsDB.BuyItem(ctx, id, &b)
	if err != nil {
		h.err(c, "restoring an item", err)
		return
	}
	h.resWithStatus(c, http.StatusOK, models.ID{ID: id})
}

func (h *itemHandler) ToggleItem(c *gin.Context) {
	ctx, _, _, def := h.start(c, log.GetFuncName())
	defer def()
	log.Logger(ctx).Info().Msg("Start")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	err = itemsDB.BuyItem(ctx, id, nil)
	if err != nil {
		h.err(c, "toggling an item", err)
		return
	}
	h.resWithStatus(c, http.StatusOK, models.ID{ID: id})
}

func (h *itemHandler) DeleteItem(c *gin.Context) {
	ctx, _, _, def := h.start(c, log.GetFuncName())
	defer def()
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	err = itemsDB.DeleteItem(ctx, id)
	if err != nil {
		h.err(c, "getting an item", err)
		return
	}
	h.resWithStatus(c, http.StatusNoContent, nil)
}

func (h *itemHandler) CreateItem(c *gin.Context) {
	ctx, _, _, def := h.start(c, log.GetFuncName())
	defer def()
	log.Logger(ctx).Info().Msg("Start")
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}
	var item models.ItemWithID
	err = c.BindJSON(&item)
	if err != nil {
		h.err(c, "BindJSON", err)
		return
	}
	item.ID = ""

	err = itemsDB.UpsertItem(ctx, &item)
	if err != nil {
		h.err(c, "UpsertItem", err)
		return
	}

	h.res(c, item)
}

func (h *itemHandler) EditItem(c *gin.Context) {
	ctx, _, _, def := h.start(c, log.GetFuncName())
	defer def()
	log.Logger(ctx).Info().Msg("Start")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}
	var item models.ItemWithID
	err = c.BindJSON(&item)
	if err != nil {
		h.err(c, "BindJSON", err)
		return
	}
	item.ID = id

	err = itemsDB.UpsertItem(ctx, &item)
	if err != nil {
		h.err(c, "UpsertItem", err)
		return
	}

	h.res(c, item)
}

type PaginationQuery struct {
	Start int    `form:"_start"`
	End   int    `form:"_end"`
	Sort  string `form:"_sort"`
	Order string `form:"_order"`
	Query string `form:"q"`
}

func (h *itemHandler) GetItems(c *gin.Context) {
	fn := log.GetFuncName()
	ctx, span, _, def := h.start(c, fn)
	defer def()

	span.AddEvent("Started")
	start := time.Now()

	var p PaginationQuery
	if err := c.ShouldBindQuery(&p); err != nil {
		h.err(c, "parsing parameters", err)
		return
	}

	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	var itemsOut []*models.ItemWithID
	var total int
	itemsOut, total, err = itemsDB.GetItems(ctx, &db.PaginationQuery{
		Start: p.Start,
		End:   p.End,
		Sort:  p.Sort,
		Order: p.Order,
		Query: p.Query,
	}, p.Query, h.boughtLast)
	if err != nil {
		h.err(c, "getting items", err)
	}
	c.Header("X-Total-Count", strconv.Itoa(total))

	end := time.Now()
	dur := end.Sub(start)
	h.apiCalls.Record(ctx, dur.Milliseconds(),
		metric.WithAttributes(
			attribute.String("label", h.label),
			attribute.String("func", fn),
		))

	h.res(c, itemsOut)
}
