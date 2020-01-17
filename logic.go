package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/HDIOES/anime-app/dao"
)

//TelegramHandler struct
type TelegramHandler struct {
	udao *dao.UserDAO
	sdao *dao.SubscriptionDAO
	adao *dao.AnimeDAO
	db   *sql.DB
}

func (th *TelegramHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	update := &Update{}
	decodeErr := decoder.Decode(update)
	if decodeErr != nil {
		//return 400 status
	}
	switch update.Message.Text {
	case "/start":
		th.startCommand(update)
	case "/animes":
		th.animesCommand(update)
	default:
		th.defaultCommand(update)
	}
}

func (th *TelegramHandler) startCommand(update *Update) error {
	telegramUserID := strconv.FormatInt(update.Message.From.ID, 10)
	telegramUsername := update.Message.From.Username
	tx, txErr := th.db.Begin()
	if txErr != nil {
		return txErr
	}
	if insertErr := th.udao.Insert(tx, telegramUserID, telegramUsername); insertErr != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return insertErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return commitErr
	}
	return nil
}

func (th *TelegramHandler) animesCommand(update *Update) error {
	return nil
}

func (th *TelegramHandler) defaultCommand(update *Update) error {
	return nil
}

//Update struct
type Update struct {
	UpdateID int64   `json:"udpate_id"`
	Message  Message `json:"message"`
}

//Message struct
type Message struct {
	MessageID int64  `json:"message_id"`
	From      User   `json:"from"`
	Text      string `json:"text"`
}

//User struct
type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}
