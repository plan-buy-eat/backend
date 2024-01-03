package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/db"
	"github.com/shoppinglist/models"
	"net/http"
	"strconv"
	"time"

	"github.com/shoppinglist/log"
)

type Handler interface {
	GetItems(c *gin.Context)
	GetItem(c *gin.Context)
	HealthZ(context *gin.Context)
	Init(context *gin.Context)
}

type handler struct {
	config *config.Config
}

func New() Handler {
	return &handler{
		config.Get(),
	}
}

func (h *handler) HealthZ(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/plain")
	itemsDB, err := db.NewDB(ctx)
	if err != nil {
		h.err(c, "getting db", err)
		return
	}
	report, err := itemsDB.Ping(ctx)
	if err != nil {
		h.err(c, "pinging db", err)
		return
	}
	t := fmt.Sprintf("%s(%s)@%s: %s\nDB:%s\n", h.config.ServiceName, h.config.HostName, h.config.ServiceVersion, time.Now().Local().Format(time.RFC1123Z), report)
	log.Logger().Printf("response %s\n", t)
	h.res(c, t)
}

func (h *handler) Init(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/plain")
	err := db.InitDB(ctx)
	if err != nil {
		h.err(c, "getting db", err)
		return
	}
	h.res(c, "OK")
}

//type ID struct {
//	ID string `uri:"id" binding:"required"`
//}

func (h *handler) GetItem(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "application/json")
	id := c.Param("id")
	if id == "" {
		h.errWithStatus(c, http.StatusBadRequest, "bad request", fmt.Errorf("no id specified"))
	}
	itemsDB, err := db.NewDB(ctx)
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

func (h *handler) GetItems(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "application/json")

	itemsDB, err := db.NewDB(ctx)
	if err != nil {
		h.err(c, "getting db", err)
		return
	}

	var itemsOut []*models.ItemWithId
	q := c.Request.URL.Query().Get("q")
	if q != "" {
		itemsOut, err = itemsDB.SearchItems(ctx, "title-index", q)
		if err != nil {
			h.err(c, "searching items", err)
		}
		c.Header("X-Total-Count", strconv.Itoa(len(itemsOut)))
		h.res(c, itemsOut)
	} else {
		itemsOut, err = itemsDB.GetItems(ctx)
		if err != nil {
			h.err(c, "getting items", err)
		}
		c.Header("X-Total-Count", strconv.Itoa(len(itemsOut)))
		h.res(c, itemsOut)
	}

	c.Status(http.StatusOK)
}

func (h *handler) err(c *gin.Context, message string, err error) {
	h.errWithStatus(c, http.StatusInternalServerError, message, err)
}

func (h *handler) errWithStatus(c *gin.Context, status int, message string, err error) {
	err = c.AbortWithError(status, fmt.Errorf("%s: %w", message, err))
	if err != nil {
		log.Logger().Fatal().Err(err).Msg("error aborting with error")
	}
}

func (h *handler) res(c *gin.Context, data any) {
	h.resWithStatus(c, http.StatusOK, data)
}

func (h *handler) resWithStatus(c *gin.Context, status int, data any) {
	var out []byte
	var err error
	if s, ok := data.(string); ok {
		out = []byte(s)
	} else {
		out, err = json.Marshal(data)
		if err != nil {
			h.err(c, "marshaling items", err)
			return
		}
	}
	_, err = c.Writer.Write(out)
	if err != nil {
		h.err(c, "writing response", err)
		return
	}
	c.Status(status)
}
