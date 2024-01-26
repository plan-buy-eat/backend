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
	"github.com/shoppinglist/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusOK)
		c.Next()
		for _, err := range c.Errors {
			log.Logger().Err(err).Msg("Error getting db")
		}
		if len(c.Errors) > 0 && c.Writer.Status() == http.StatusOK {
			c.JSON(http.StatusInternalServerError, "Internal Server Error")
		}
	}
}

func main() {
	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// log.Logger().Info().Any("env", os.Environ()).Msgf("Env")

	c := config.Get()
	t := fmt.Sprintf("%s(%s)@%s: %s\n", c.ServiceName, c.HostName, c.ServiceVersion, time.Now().Local().Format(time.RFC1123Z))
	log.Logger().Info().Msgf("Starting %s\n", t)

	tp, err := initTracer()
	if err != nil {
		log.Logger().Fatal().Err(err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Logger().Err(err).Msg("Error shutting down tracer provider")
		}
	}()

	port := config.Get().Port
	listenAddress := "0.0.0.0:" + port
	log.Logger().Info().Msgf("Listening at %s", listenAddress)

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
	r.Use(ErrorHandler())

	r.Use(otelgin.Middleware(config.Get().ServiceName))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, "Page not found")
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, "Method not found")
	})

	genericHandler := handlers.NewGenericHandler()
	r.GET("/init", genericHandler.Init)
	r.GET("/healthz", genericHandler.HealthZ)

	toBuyHandler := handlers.NewItemHandler(sql.NullBool{
		Bool:  false,
		Valid: true,
	})
	toBuy := r.Group("/tobuy")
	toBuy.GET("", toBuyHandler.GetItems)
	toBuy.GET("/:id", toBuyHandler.GetItem)
	toBuy.DELETE("/:id", toBuyHandler.BuyItem)

	boughtHandler := handlers.NewItemHandler(sql.NullBool{
		Bool:  true,
		Valid: true,
	})
	bought := r.Group("/bought")
	bought.GET("", boughtHandler.GetItems)
	bought.GET("/:id", boughtHandler.GetItem)
	bought.DELETE("/:id", boughtHandler.RestoreItem)

	srv := &http.Server{
		Addr:    listenAddress,
		Handler: r,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Logger().Fatal().Err(err).Msg("listen\n")
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
	log.Logger().Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Logger().Fatal().Err(err).Msg("server shutdown")
	}
	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Logger().Info().Msg("timeout of 5 seconds.")
	}
	log.Logger().Info().Msg("server is stopped")
}

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}
