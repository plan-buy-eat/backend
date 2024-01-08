package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/shoppinglist/config"
	"github.com/shoppinglist/item-service/handlers"
	"github.com/shoppinglist/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	//zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger().Info().Any("env", os.Environ()).Msgf("Env")

	port := config.Get().Port
	listenAddress := "0.0.0.0:" + port
	log.Logger().Printf("Listening at %s", listenAddress)

	router := gin.Default()
	router.HandleMethodNotAllowed = true
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://shoppinglist.turevskiy.kharkiv.ua"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "X-Total-Count"},
		AllowCredentials: true,
		//AllowOriginFunc: func(origin string) bool {
		//	return origin == "https://github.com"
		//},
		MaxAge: 12 * time.Hour,
	}))
	router.Use(ErrorHandler())
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, "Page not found")
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, "Method not found")
	})

	genericHandler := handlers.NewGenericHandler()
	router.GET("/init", genericHandler.Init)
	router.GET("/healthz", genericHandler.HealthZ)

	itemHandler := handlers.NewItemHandler(sql.NullBool{
		Bool:  false,
		Valid: true,
	})

	items := router.Group("/items")
	items.GET("", itemHandler.GetItems)
	items.GET("/:id", itemHandler.GetItem)
	items.DELETE("/:id", itemHandler.BuyItem)

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
