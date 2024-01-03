package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/item-service/handler"
	"github.com/shoppinglist/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

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
	//zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger().Info().Any("env", os.Environ()).Msgf("Env")

	port := config.Get().Port
	listenAddress := ":" + port
	log.Logger().Printf("Listening at %s", listenAddress)

	router := gin.Default()
	router.Use(CORSMiddleware())
	router.Use(ErrorHandler())

	h := handler.New()
	items := router.Group("/items")
	items.GET("/", h.GetItems)
	items.GET("/:id", h.GetItem)
	router.GET("/init", h.Init)
	router.GET("/healthz", h.HealthZ)

	srv := &http.Server{
		Addr:    listenAddress,
		Handler: router,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Logger().Fatal().Err(err).Msg("listen\n")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Logger().Info().Msg("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Logger().Fatal().Err(err).Msg("Server Shutdown")
	}
	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Logger().Info().Msg("timeout of 5 seconds.")
	}
	log.Logger().Info().Msg("Server exiting")
}
