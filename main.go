package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/HDIOES/anime-app/dao"
	_ "github.com/lib/pq"
	nats "github.com/nats-io/nats.go"
	migrate "github.com/rubenv/sql-migrate"
	"go.uber.org/dig"
)

func main() {
	container := dig.New()
	container.Provide(func() *Settings {
		if jsonFile, openErr := os.Open("settings.json"); openErr != nil {
			panic(openErr)
		} else {
			defer jsonFile.Close()
			decoder := json.NewDecoder(jsonFile)
			settings := &Settings{}
			if decodeErr := decoder.Decode(settings); decodeErr != nil {
				panic(decodeErr)
			} else {
				return settings
			}
		}
	})
	container.Provide(func(settings *Settings) (*sql.DB, *nats.Conn, *dao.AnimeDAO, *dao.UserDAO, *dao.SubscriptionDAO) {
		db, err := sql.Open("postgres", settings.DatabaseURL)
		if err != nil {
			panic(err)
		}
		db.SetMaxIdleConns(settings.MaxIdleConnections)
		db.SetMaxOpenConns(settings.MaxOpenConnections)
		timeout := strconv.Itoa(settings.ConnectionTimeout) + "s"
		timeoutDuration, durationErr := time.ParseDuration(timeout)
		if durationErr != nil {
			defer db.Close()
			panic(durationErr)
		} else {
			db.SetConnMaxLifetime(timeoutDuration)
		}
		if n, migrateErr := migrate.Exec(db, "postgres", &migrate.FileMigrationSource{Dir: settings.MigrationPath}, migrate.Up); migrateErr != nil {
			panic(migrateErr)
		} else {
			log.Printf("Applied %d migrations!\n", n)
		}
		natsConnection, ncErr := nats.Connect(settings.NatsURL)
		if ncErr != nil {
			panic(ncErr)
		}
		return db, natsConnection, &dao.AnimeDAO{Db: db}, &dao.UserDAO{Db: db}, &dao.SubscriptionDAO{Db: db}
	})
	container.Invoke(func(db *sql.DB, settings *Settings, natsConnection *nats.Conn, adao *dao.AnimeDAO, udao *dao.UserDAO, sdao *dao.SubscriptionDAO) {
		srv := &http.Server{Addr: ":8000", Handler: &TelegramHandler{
			db:             db,
			udao:           udao,
			sdao:           sdao,
			adao:           adao,
			natsConnection: natsConnection,
			settings:       settings,
		}}
		log.Fatal(srv.ListenAndServe())
	})
}

//Settings mapping object for settings.json
type Settings struct {
	DatabaseURL        string `json:"databaseUrl"`
	MaxOpenConnections int    `json:"maxOpenConnections"`
	MaxIdleConnections int    `json:"maxIdleConnections"`
	ConnectionTimeout  int    `json:"connectionTimeout"`
	ApplicationPort    int    `json:"port"`
	MigrationPath      string `json:"migrationPath"`
	NatsURL            string `json:"natsUrl"`
	NatsSubject        string `json:"natsSubject"`
}
