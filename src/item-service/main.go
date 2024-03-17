package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/xid"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/item-service/handlers"
	"github.com/shoppinglist/item-service/otel"
	"github.com/shoppinglist/log"

	// "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	//// Handle SIGINT (CTRL+C) gracefully.
	//ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	//defer stop()
	ctx := context.Background()

	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger(ctx).Info().Any("env", os.Environ()).Msgf("Env")

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	c := config.Get(ctx)
	t := fmt.Sprintf("%s(%s)@%s: %s\n", c.ServiceName, c.HostName, c.ServiceVersion, time.Now().Local().Format(time.RFC1123Z))
	log.Logger(ctx).Info().Msgf("Starting %s\n", t)

	// Set up OpenTelemetry.
	tracer, meter, otelShutdown, err := otel.SetupOTelSDK(ctx, c)
	if err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("SetupOTelSDK")
		time.Sleep(time.Minute)
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
		if err != nil {
			log.Logger(ctx).Fatal().Err(err).Msg("otelShutdown")
		}
	}()
	c.Tracer = tracer
	c.Meter = meter

	port := c.Port
	listenAddress := "0.0.0.0:" + port
	log.Logger(ctx).Info().Msgf("Listening at %s", listenAddress)

	r := gin.New()
	r.Use(
		requestid.New(
			requestid.WithGenerator(func() string {
				return xid.New().String()
			}),
		),
	)
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, "requestId", requestid.Get(c))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	pprof.Register(r)
	r.HandleMethodNotAllowed = true
	r.Use(LoggerMiddleware)
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://shoppinglist.turevskiy.kharkiv.ua"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
		//AllowOriginFunc: func(origin string) bool {
		//	return origin == "https://github.com"
		//},
		MaxAge: 12 * time.Hour,
	}))
	r.Use(handlers.ErrorHandler())

	// r.Use(otelgin.Middleware(c.ServiceName))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, "Page not found")
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, "Method not found")
	})

	genericHandler := handlers.NewGenericHandler(ctx)
	r.GET("/init", genericHandler.Init)
	r.GET("/healthz", genericHandler.HealthZ)

	toBuyHandler, err := handlers.NewItemHandler(ctx,
		"toBuy",
		sql.NullBool{
			Bool:  false,
			Valid: true,
		},
		false,
	)
	if err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("ToBuyHandler")
	}
	toBuy := r.Group("/tobuy")
	toBuy.GET("", toBuyHandler.GetItems)
	toBuy.GET("/:id", toBuyHandler.GetItem)
	toBuy.DELETE("/:id", toBuyHandler.BuyItem)

	boughtHandler, err := handlers.NewItemHandler(ctx,
		"bought",
		sql.NullBool{
			Bool:  true,
			Valid: true,
		},
		false,
	)
	if err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("BoughtHandler")
	}
	bought := r.Group("/bought")
	bought.GET("", boughtHandler.GetItems)
	bought.GET("/:id", boughtHandler.GetItem)
	bought.DELETE("/:id", boughtHandler.RestoreItem)

	allItemsHandler, err := handlers.NewItemHandler(ctx,
		"allItems",
		sql.NullBool{
			Valid: false,
		},
		true,
	)
	if err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("AllItemsHandler")
	}
	allItems := r.Group("/items")
	allItems.GET("", allItemsHandler.GetItems)
	allItems.GET("/:id", allItemsHandler.GetItem)
	allItems.DELETE("/:id", allItemsHandler.ToggleItem)

	inventoryHandler, err := handlers.NewItemHandler(ctx,
		"allItems",
		sql.NullBool{
			Valid: false,
		},
		false,
	)
	if err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("AllItemsHandler")
	}
	inventory := r.Group("/inventory")
	inventory.GET("", inventoryHandler.GetItems)
	inventory.GET("/:id", inventoryHandler.GetItem)
	inventory.PUT("/:id", inventoryHandler.EditItem)
	inventory.POST("", inventoryHandler.CreateItem)
	inventory.DELETE("/:id", inventoryHandler.DeleteItem)

	srv := &http.Server{
		Addr:    listenAddress,
		Handler: r,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Logger(ctx).Fatal().Err(err).Msg("listen\n")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Logger(ctx).Info().Msg("shutting down server...")

	ctx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctx); err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("server shutdown")
	}
	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Logger(ctx).Info().Msg("timeout of 5 seconds.")
	}
	log.Logger(ctx).Info().Msg("server is stopped")
}

func LoggerMiddleware(c *gin.Context) {
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery
	if raw != "" {
		path = path + "?" + raw
	}
	start := time.Now()

	c.Next()

	end := time.Now()
	latency := end.Sub(start)

	ctx := c.Request.Context()
	log.Logger(ctx).Info().
		Int("status", c.Writer.Status()).
		Str("method", c.Request.Method).
		Str("path", path).
		Str("ip", c.ClientIP()).
		Dur("latency", latency).
		Str("user_agent", c.Request.UserAgent()).
		Int("body_size", c.Writer.Size()).
		Msg("request processed")
}
