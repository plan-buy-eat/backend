package handlers

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/db"
	"github.com/shoppinglist/models"
	"net/http"
	"strconv"
)

type ItemHandler interface {
	GetItems(c *gin.Context)
	GetItem(c *gin.Context)
	BuyItem(c *gin.Context)
}

type itemHandler struct {
	genericHandler
	bought sql.NullBool
}

func NewItemHandler(bought sql.NullBool) ItemHandler {
	return &itemHandler{
		genericHandler{
			config: config.Get(),
		},
		bought,
	}
}

func (h *itemHandler) GetItem(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "application/json")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewItemsDB(ctx, h.bought)
	if err != nil {
		h.err(c, "getting db", err)
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
	itemsDB, err := db.NewItemsDB(ctx, h.bought)
	if err != nil {
		h.err(c, "getting db", err)
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
	itemsDB, err := db.NewItemsDB(ctx, h.bought)
	if err != nil {
		h.err(c, "getting db", err)
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
//	itemsDB, err := db.NewItemsDB(ctx, h.bought)
//	if err != nil {
//		h.err(c, "getting db", err)
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
	ctx := c.Request.Context()

	var p PaginationQuery
	if err := c.ShouldBindQuery(&p); err != nil {
		h.err(c, "parsing parameters", err)
		return
	}

	c.Header("Content-Type", "application/json")
	itemsDB, err := db.NewItemsDB(ctx, h.bought)
	if err != nil {
		h.err(c, "getting db", err)
		return
	}

	var itemsOut []*models.ItemWithID
	var total int
	if p.Query != "" {
		itemsOut, err = itemsDB.SearchItems(ctx, p.Query)
		if err != nil {
			h.err(c, "searching items", err)
		}
		c.Header("X-Total-Count", strconv.Itoa(len(itemsOut)))
		h.res(c, itemsOut)
	} else {
		itemsOut, total, err = itemsDB.GetItems(ctx, &db.PaginationQuery{
			Start: p.Start,
			End:   p.End,
			Sort:  p.Sort,
			Order: p.Order,
			Query: p.Query,
		})
		if err != nil {
			h.err(c, "getting items", err)
		}
		c.Header("X-Total-Count", strconv.Itoa(total))
		h.res(c, itemsOut)
	}

	c.Status(http.StatusOK)
}
