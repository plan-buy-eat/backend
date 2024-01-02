package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/shoppinglist/log"
	"github.com/shoppinglist/models"
	"strconv"

	"github.com/shoppinglist/db"
	"github.com/shoppinglist/utils/config"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var serviceName string
var hostName string
var serviceVersion string

func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Add("Access-Control-Expose-Headers", "X-Total-Count")

		if r.Method == "OPTIONS" {
			http.Error(w, "No Content", http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func main() {
	serviceName = os.Getenv("SERVICE_NAME")
	hostName = os.Getenv("HOSTNAME")
	serviceVersion = os.Getenv("SERVICE_VERSION")
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger().Info().Any("env", os.Environ()).Msgf("Env")

	http.HandleFunc("/", CORS(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var out []byte
		if r.URL.Path == "/items" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")

			itemsDB, err := db.NewDB(ctx)
			if err != nil {
				log.Logger().Err(err).Msg("Error searching for items")
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}

			id := r.URL.Query().Get("id")
			if id != "" {
				itemOut, err := itemsDB.GetItem(ctx, id)
				if err != nil {
					log.Logger().Err(err).Msg("Error searching for items")
					http.Error(w, "Server Error", http.StatusInternalServerError)
					return
				}
				out, err = json.Marshal(itemOut)
				if err != nil {
					log.Logger().Err(err).Msg("Error marshaling items")
					http.Error(w, "Server Error", http.StatusInternalServerError)
					return
				}
			}

			var itemsOut []*models.ItemWithId
			q := r.URL.Query().Get("q")
			if q != "" {
				itemsOut, err = itemsDB.SearchItems(ctx, "title-index", q)
				if err != nil {
					log.Logger().Err(err).Msg("Error searching items")
					http.Error(w, "Server Error", http.StatusInternalServerError)
				}
				out, err = json.Marshal(itemsOut)
				if err != nil {
					log.Logger().Err(err).Msg("Error marshaling items")
					http.Error(w, "Server Error", http.StatusInternalServerError)
				}
			} else {
				itemsOut, err = itemsDB.GetItems(ctx)
				if err != nil {
					log.Logger().Err(err).Msg("Error getting items")
					http.Error(w, "Server Error", http.StatusInternalServerError)
				}
				out, err = json.Marshal(itemsOut)
				if err != nil {
					log.Logger().Err(err).Msg("Error marshaling items")
					http.Error(w, "Server Error", http.StatusInternalServerError)
				}
			}
			w.Header().Set("X-Total-Count", strconv.Itoa(len(itemsOut)))

			_, err = w.Write(out)
			if err != nil {
				log.Logger().Err(err).Msg("Error writing response")
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}
		} else if r.URL.Path == "/init" && r.Method == "GET" {
			err := db.InitDB(ctx)
			if err != nil {
				log.Logger().Err(err).Msg("Error searching for items")
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			_, err = w.Write([]byte("OK\n"))
			if err != nil {
				log.Logger().Err(err).Msg("Error writing response")
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}
		} else if r.URL.Path == "/healthz" && r.Method == "GET" {
			w.Header().Set("Content-Type", "text/plain")
			t := fmt.Sprintf("%s(%s)@%s: %s\n", serviceName, hostName, serviceVersion, time.Now().Local().Format(time.RFC1123Z))
			log.Logger().Printf("response %s\n", t)
			_, err := w.Write([]byte(t + "\n"))
			if err != nil {
				log.Logger().Err(err).Msg("Error writing response")
				http.Error(w, "Server Error", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
	}))

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
