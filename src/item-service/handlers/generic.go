package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/db"

	"github.com/shoppinglist/log"
)

type GenericHandler interface {
	HealthZ(context *gin.Context)
	Init(context *gin.Context)
}

type genericHandler struct {
	config *config.Config
}

func NewGenericHandler() GenericHandler {
	return &genericHandler{
		config.Get(),
	}
}

func (h *genericHandler) HealthZ(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/plain")
	itemsDB, err := db.NewGenericDB(ctx, h.config)
	if err != nil {
		h.err(c, "NewGenericDB", err)
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

func (h *genericHandler) Init(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/plain")
	err := db.InitDB(ctx, h.config)
	if err != nil {
		h.err(c, "init db", err)
		return
	}
	h.res(c, "OK")
}

func (h *genericHandler) err(c *gin.Context, message string, err error) {
	h.errWithStatus(c, http.StatusInternalServerError, message, err)
}

func (h *genericHandler) errWithStatus(c *gin.Context, status int, message string, err error) {
	err = c.AbortWithError(status, fmt.Errorf("%s: %w", message, err))
	if err != nil {
		log.Logger().Error().Err(err).Msg("error aborting with error")
		c.Status(500)
	}
}

func (h *genericHandler) res(c *gin.Context, data any) {
	h.resWithStatus(c, http.StatusOK, data)
}

func (h *genericHandler) resWithStatus(c *gin.Context, status int, data any) {
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
