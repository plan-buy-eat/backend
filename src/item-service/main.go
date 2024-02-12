package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
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
		log.Logger(ctx).Fatal().Err(err).Msg("otelShutdown")
	}()
	c.Tracer = tracer
	c.Meter = meter

	port := c.Port
	listenAddress := "0.0.0.0:" + port
	log.Logger(ctx).Info().Msgf("Listening at %s", listenAddress)

	r := gin.New()
	pprof.Register(r)
	r.HandleMethodNotAllowed = true
	r.Use(logger.SetLogger(
		logger.WithLogger(func(_ *gin.Context, l zerolog.Logger) zerolog.Logger {
			return l.Output(gin.DefaultWriter).With().Logger()
		}),
	))
	r.Use(gin.Recovery())
	r.Use(
		requestid.New(
			requestid.WithGenerator(func() string {
				return xid.New().String()
			}),
		),
	)
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
		})
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
		})
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
		})
	if err != nil {
		log.Logger(ctx).Fatal().Err(err).Msg("AllItemsHandler")
	}
	allItems := r.Group("/items")
	allItems.GET("", allItemsHandler.GetItems)
	allItems.GET("/:id", allItemsHandler.GetItem)
	allItems.DELETE("/:id", allItemsHandler.RestoreItem)

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
