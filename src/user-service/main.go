package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/couchbase/gocb/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/shoppinglist/log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var serviceName string
var port = "80"

func main() {
	serviceName = os.Getenv("SERVICE_NAME")
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger().Info().Any("env", os.Environ()).Msgf("Env")

	c := func() {
		// Uncomment following line to enable logging
		//gocb.SetLogger(gocb.VerboseStdioLogger())

		// Update this to your cluster details
		// For a secure cluster connection, use `couchbases://<your-cluster-ip>` instead.
		connectionString := os.Getenv("COUCHBASE_CONNECTION_STRING")
		//connectionString := "couchbase://127.0.0.1?network=external"
		//connectionString := "127.0.0.1?network=external"
		bucketName := os.Getenv("COUCHBASE_BUCKET")
		username := os.Getenv("COUCHBASE_USERNAME")
		password := os.Getenv("COUCHBASE_PASSWORD")
		fmt.Print(connectionString, bucketName, username, password)

		cluster, err := gocb.Connect(connectionString, gocb.ClusterOptions{
			Authenticator: gocb.PasswordAuthenticator{
				Username: username,
				Password: password,
			},
		})
		if err != nil {
			log.Logger().Err(err)
			return
		}

		bucket := cluster.Bucket(bucketName)

		err = bucket.WaitUntilReady(5*time.Second, nil)
		if err != nil {
			log.Logger().Err(err)
			return

		}

		// Get a reference to the default collection, required for older Couchbase server versions
		// col := bucket.DefaultCollection()

		// TODO: create scope and collections if not exists
		col := bucket.Scope("0").Collection("users")

		type User struct {
			Name      string   `json:"name"`
			Email     string   `json:"email"`
			Interests []string `json:"interests"`
		}

		// Create and store a Document
		_, err = col.Upsert("u:jade",
			User{
				Name:      "Jade",
				Email:     "jade@test-email.com",
				Interests: []string{"Swimming", "Rowing"},
			}, nil)
		if err != nil {
			log.Logger().Err(err)
			return

		}

		// Get the document back
		getResult, err := col.Get("u:jade", nil)
		if err != nil {
			log.Logger().Err(err)
			return

		}

		var inUser User
		err = getResult.Content(&inUser)
		if err != nil {
			log.Logger().Err(err)
			return

		}
		fmt.Printf("User: %v\n", inUser)
		//
		//// Perform a N1QL Query
		//inventoryScope := bucket.Scope("inventory")
		//queryResult, err := inventoryScope.Query(
		//	fmt.Sprintf("SELECT * FROM airline WHERE id=10"),
		//	&gocb.QueryOptions{Adhoc: true},
		//)
		//if err != nil {
		//	log.Print(err)
		//	return
		//
		//}
		//
		//// Print each found Row
		//for queryResult.Next() {
		//	var result interface{}
		//	err := queryResult.Row(&result)
		//	if err != nil {
		//		log.Print(err)
		//		return
		//
		//	}
		//	fmt.Print(result)
		//}
		//
		//if err := queryResult.Err(); err != nil {
		//	log.Print(err)
		//	return
		//
		//}
	}
	_ = c
	c()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Logger().Info().Msgf("HTTP %s %s%s\n", r.Method, r.Host, r.URL)
		// ctx := r.Context()

		if r.URL.Path == "/healthz" && r.Method == "GET" {
			w.Header().Set("Content-Type", "text/plain")
			t := fmt.Sprintf("%s: %s\n", serviceName, time.Now().Local().Format(time.RFC1123Z))
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
	})

	listenAddress := fmt.Sprintf(":%s", port)
	log.Logger().Info().Msgf("Listening at %s", listenAddress)

	httpServer := http.Server{
		Addr: listenAddress,
	}

	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := httpServer.Shutdown(context.Background()); err != nil {
			log.Logger().Info().Msgf("HTTP Server Shutdown Error: %v", err)
		}
		close(idleConnectionsClosed)
	}()

	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Logger().Fatal().Msgf("HTTP server ListenAndServe Error")
	}

	<-idleConnectionsClosed

	log.Logger().Info().Msgf("Bye bye")
}
