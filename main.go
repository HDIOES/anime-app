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
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	migrate "github.com/rubenv/sql-migrate"
	"go.uber.org/dig"

	_ "github.com/lib/pq"
)

const (
	databaseURLEnvName               = "DATABASE_URL"
	maxOpenConnectionsEnvName        = "MAX_OPEN_CONNECTIONS"
	maxIdleConnectionsEnvName        = "MAX_IDLE_CONNECTIONS"
	databaseConnectionTimeoutEnvName = "DATABASE_CONNECTION_TIMEOUT"
	applicationPortEnvName           = "PORT"
	migrationPathEnvName             = "DATABASE_MIGRATION_PATH"
	natsURLEnvName                   = "NATS_URL"
	natsSubjectEnvName               = "NATS_SUBJECT"
)

func main() {
	container := dig.New()
	container.Provide(func() *Settings {
		if jsonFile, openErr := os.Open("settings.json"); openErr != nil {
			log.Panicln(openErr)
		} else {
			defer jsonFile.Close()
			decoder := json.NewDecoder(jsonFile)
			settings := &Settings{}
			if decodeErr := decoder.Decode(settings); decodeErr != nil {
				log.Panicln(decodeErr)
			} else {
				setSettingsFromEnv(settings)
				return settings
			}
		}
		panic("Unreachable code")
	})
	container.Provide(func(settings *Settings) (*sql.DB, *nats.Conn, *dao.AnimeDAO, *dao.UserDAO, *dao.SubscriptionDAO) {
		db, err := sql.Open("postgres", settings.DatabaseURL)
		if err != nil {
			log.Panicln(err)
		}
		db.SetMaxIdleConns(settings.MaxIdleConnections)
		db.SetMaxOpenConns(settings.MaxOpenConnections)
		timeout := strconv.Itoa(settings.ConnectionTimeout) + "s"
		timeoutDuration, durationErr := time.ParseDuration(timeout)
		if durationErr != nil {
			defer db.Close()
			log.Panicln(durationErr)
		} else {
			db.SetConnMaxLifetime(timeoutDuration)
		}
		if n, migrateErr := migrate.Exec(db, "postgres", &migrate.FileMigrationSource{Dir: settings.MigrationPath}, migrate.Up); migrateErr != nil {
			log.Panicln(migrateErr)
		} else {
			log.Printf("Applied %d migrations!\n", n)
		}
		natsConnection, ncErr := nats.Connect(settings.NatsURL)
		if ncErr != nil {
			log.Panicln(ncErr)
		}
		return db, natsConnection, &dao.AnimeDAO{Db: db}, &dao.UserDAO{Db: db}, &dao.SubscriptionDAO{Db: db}
	})
	container.Invoke(func(settings *Settings, natsConnection *nats.Conn, adao *dao.AnimeDAO, udao *dao.UserDAO, sdao *dao.SubscriptionDAO) {
		handler := &TelegramHandler{
			udao:           udao,
			sdao:           sdao,
			adao:           adao,
			natsConnection: natsConnection,
			settings:       settings,
		}
		/*notification := Notification{
			Type: "setWebhookNotification",
		}
		if err := handler.sendNotification(notification); err != nil {
			log.Panicln(err)
		}*/
		srv := &http.Server{Addr: ":" + strconv.Itoa(settings.ApplicationPort), Handler: handler}
		log.Fatal(srv.ListenAndServe())
	})
}

func setSettingsFromEnv(settings *Settings) {
	if value := os.Getenv(databaseURLEnvName); value != "" {
		settings.DatabaseURL = value
	}
	if value := os.Getenv(maxOpenConnectionsEnvName); value != "" {
		if intValue, err := strconv.Atoi(value); err != nil {
			log.Panicln(err)
		} else {
			settings.MaxOpenConnections = intValue
		}
	}
	if value := os.Getenv(maxIdleConnectionsEnvName); value != "" {
		if intValue, err := strconv.Atoi(value); err != nil {
			log.Panicln(err)
		} else {
			settings.MaxIdleConnections = intValue
		}
	}
	if value := os.Getenv(databaseConnectionTimeoutEnvName); value != "" {
		if intValue, err := strconv.Atoi(value); err != nil {
			log.Panicln(err)
		} else {
			settings.ConnectionTimeout = intValue
		}
	}
	if value := os.Getenv(applicationPortEnvName); value != "" {
		if intValue, err := strconv.Atoi(value); err != nil {
			log.Panicln(err)
		} else {
			settings.ApplicationPort = intValue
		}
	}
	if value := os.Getenv(migrationPathEnvName); value != "" {
		settings.MigrationPath = value
	}
	if value := os.Getenv(natsURLEnvName); value != "" {
		settings.NatsURL = value
	}
	if value := os.Getenv(natsSubjectEnvName); value != "" {
		settings.NatsSubject = value
	}
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

//StackTracer struct
type StackTracer interface {
	StackTrace() errors.StackTrace
}

//HandleError func
func HandleError(handledErr error) {
	if err, ok := handledErr.(StackTracer); ok {
		for _, f := range err.StackTrace() {
			log.Printf("%+s:%d\n", f, f)
		}
	} else {
		log.Println("Unknown error: ", err)
	}
}
