package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	label  string
}

func NewGenericHandler(ctx context.Context) GenericHandler {
	return &genericHandler{
		config.Get(ctx),
		"generic",
	}
}

func (h *genericHandler) HealthZ(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/plain")
	genericDB, err := db.NewGenericDB(ctx, "generic", h.config)
	if err != nil {
		h.err(c, "NewGenericDB", err)
		return
	}
	report, err := genericDB.Ping(ctx)
	if err != nil {
		h.err(c, "pinging db", err)
		return
	}
	t := fmt.Sprintf("%s(%s)@%s: %s\nDB:%s\n", h.config.ServiceName, h.config.HostName, h.config.ServiceVersion, time.Now().Local().Format(time.RFC1123Z), report)
	log.Logger(ctx).Printf("response %s\n", t)
	h.res(c, t)
}

func (h *genericHandler) Init(c *gin.Context) {
	ctx := c.Request.Context()
	c.Header("Content-Type", "text/plain")
	itemsDB, err := db.NewItemsDB(ctx, h.config, sql.NullBool{
		Valid: false,
	})
	if err != nil {
		h.err(c, "NewItemsDB", err)
		return
	}
	err = itemsDB.InitDB(ctx)
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

func (h *genericHandler) start(c *gin.Context, funcName string) (context.Context, trace.Span, zerolog.Logger, func()) {
	ctx := c.Request.Context()
	ctx, span := h.config.Tracer.Start(ctx, funcName)
	span.SetAttributes(attribute.String("handlerLabel", h.label))
	span.AddEvent("Started")
	l := log.Logger(ctx).With().Any("handlerLabel", h.label).Logger()

	f := func() {
		span.End()
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			span.AddEvent("Failed")
		} else {
			span.AddEvent("Succeeded")
		}
	}
	return ctx, span, l, f
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		c.Header("Content-Type", "application/json")
		c.Status(http.StatusOK)
		c.Next()
		for _, err := range c.Errors {
			log.Logger(ctx).Err(err).Msg("error while processing request")
		}
		if len(c.Errors) > 0 && c.Writer.Status() == http.StatusOK {
			c.JSON(http.StatusInternalServerError, "Internal Server Error")
		}
	}
}
