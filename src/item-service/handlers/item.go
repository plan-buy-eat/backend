package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/db"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type ItemHandler interface {
	GetItems(c *gin.Context)
	GetItem(c *gin.Context)
	BuyItem(c *gin.Context)
	RestoreItem(c *gin.Context)
}

type itemHandler struct {
	genericHandler
	bought     sql.NullBool
	tracer     trace.Tracer
	meter      metric.Meter
	apiCounter metric.Int64Counter
}

func NewItemHandler(bought sql.NullBool) (ItemHandler, error) {
	h := &itemHandler{
		genericHandler: genericHandler{
			config: config.Get(),
		},
		bought: bought,
		tracer: otel.GetTracerProvider().Tracer("ItemHandler"),
		meter:  otel.Meter("ItemHandler"),
	}

	var err error
	h.apiCounter, err = h.meter.Int64Counter(
		"api.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *itemHandler) GetItem(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "application/json")
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
	ctx := c.Request.Context()
	c.Header("Content-Type", "application/json")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	err = itemsDB.BuyItem(ctx, id, true)
	if err != nil {
		h.err(c, "buying an item", err)
		return
	}
	h.resWithStatus(c, http.StatusOK, models.ID{ID: id})
}

func (h *itemHandler) RestoreItem(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "application/json")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}

	err = itemsDB.BuyItem(ctx, id, false)
	if err != nil {
		h.err(c, "restoring an item", err)
		return
	}
	h.resWithStatus(c, http.StatusOK, models.ID{ID: id})
}

//func (h *itemHandler) DeleteItem(c *gin.Context) {
//	ctx := c.Request.Context()
//	c.Header("Content-Type", "application/json")
//	id := c.Param("id")
//	if id == "" {
//		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
//	}
//	itemsDB, err := db.NewItemsDB(ctx, h.config, h.bought)
//	if err != nil {
//		h.err(c, "NewItemsDB", err)
//		return
//	}
//
//	err = itemsDB.DeleteItem(ctx, id)
//	if err != nil {
//		h.err(c, "getting an item", err)
//		return
//	}
//	h.resWithStatus(c, http.StatusNoContent, nil)
//}

type PaginationQuery struct {
	Start int    `form:"_start"`
	End   int    `form:"_end"`
	Sort  string `form:"_sort"`
	Order string `form:"_order"`
	Query string `form:"q"`
}

func (h *itemHandler) GetItems(c *gin.Context) {
	rCtx := c.Request.Context()
	log.Logger().Info().Msg("GetItemsLog")
	ctx, span := h.tracer.Start(rCtx, "GetItemsSpan")
	defer span.End()
	defer func() {
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			span.AddEvent("Failed")
		}
	}()
	log.Logger().Info().Any("id", span.SpanContext().TraceID()).Msg("Span")

	h.apiCounter.Add(ctx, 1)

	span.AddEvent("Started")

	var p PaginationQuery
	if err := c.ShouldBindQuery(&p); err != nil {
		h.err(c, "parsing parameters", err)
		return
	}

	c.Header("Content-Type", "application/json")
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
	}, p.Query)
	if err != nil {
		h.err(c, "getting items", err)
	}
	c.Header("X-Total-Count", strconv.Itoa(total))
	h.res(c, itemsOut)

	span.AddEvent("Succeeded")

	c.Status(http.StatusOK)
}
