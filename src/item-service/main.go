package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/shoppinglist/log"

	"github.com/shoppinglist/db"
	"github.com/shoppinglist/models"
	"github.com/shoppinglist/utils/config"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var serviceName string

func main() {
	serviceName = os.Getenv("SERVICE_NAME")
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Logger().Info().Msgf("HTTP %s %s%s\n", r.Method, r.Host, r.URL)

		if r.URL.Path == "/init" && r.Method == "GET" {
			err := initDB()
			if err != nil {

			}
		} else if r.URL.Path == "/healthz" && r.Method == "GET" {
			//w.Header().Set("Content-Type", "text/plain")
			t := fmt.Sprintf("%s: %s\n", serviceName, time.Now().Local().Format(time.RFC1123Z))
			log.Logger().Printf("response %s\n", t)
			_, err := w.Write([]byte(t + "\n"))
			if err != nil {
				log.Logger().Printf("Error writing response: %v", err)
			}
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
	})

	port := config.GetValue("PORT", "80")
	listenAddress := ":" + port
	log.Logger().Printf("Listening at %s", listenAddress)

	httpServer := http.Server{
		Addr: listenAddress,
	}

	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := httpServer.Shutdown(context.Background()); err != nil {
			log.Logger().Printf("HTTP Server Shutdown Error: %v", err)
		}
		close(idleConnectionsClosed)
	}()

	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Logger().Fatal().Msgf("HTTP server ListenAndServe Error: %v", err)
	}

	<-idleConnectionsClosed

	log.Logger().Printf("Bye bye")
}

func initDB() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	itemsDB, err := db.NewDB(ctx)
	if err != nil {
		log.Logger().Error().Err(err)
		return
	}
	item1 := &models.Item{
		Title:  "Cottage Cheese",
		Amount: 1,
		Unit:   "pc",
		Bought: false,
	}
	item2 := &models.Item{
		Title:  "Avocado",
		Amount: 2,
		Unit:   "pc",
		Bought: true,
	}

	itemId1, err := itemsDB.UpsertItem(ctx, db.Key("item", item1.Title), item1)
	if err != nil {
		log.Logger().Error().Err(err)
		return err
	}
	_, err = itemsDB.UpsertItem(ctx, db.Key("item", item2.Title), item2)
	if err != nil {
		log.Logger().Error().Err(err)
		return err
	}

	item1out, err := itemsDB.GetItem(ctx, itemId1)
	if err != nil {
		log.Logger().Error().Err(err)
		return err
	}
	spew.Dump(item1out)

	itemsOut, err := itemsDB.GetAllItems(ctx)
	if err != nil {
		log.Logger().Error().Err(err)
		return err
	}
	spew.Dump(itemsOut)

	itemSearchResults, err := itemsDB.SearchItems(ctx, "title-index", "Avocado")
	if err != nil {
		log.Logger().Error().Err(err)
		return err
	}
	spew.Dump(itemSearchResults)
	return
}
