package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/nats-io/nats.go"

	"github.com/HDIOES/anime-app/dao"
)

const (
	welcomeText          = "Данный бот предназначен для своевременного уведомления о выходе в эфир эпизодов ваших любимых аниме-сериалов"
	startCommand         = "startCommand"
	animesText           = "Список сериалов"
	animesCommand        = "animesCommand"
	subscriptionsText    = "Список подписок"
	subscriptionsCommand = "subscriptionsCommand"
	defaultCommand       = "defaultCommand"
)

//TelegramHandler struct
type TelegramHandler struct {
	udao           *dao.UserDAO
	sdao           *dao.SubscriptionDAO
	adao           *dao.AnimeDAO
	db             *sql.DB
	natsConnection *nats.Conn
	settings       *Settings
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
	case "/subscriptions":
		th.subscriptionsCommand(update)
	default:
		th.defaultCommand(update)
	}
}

func (th *TelegramHandler) sendNotification(notification Notification) error {
	data, dataErr := json.Marshal(notification)
	if dataErr != nil {
		return dataErr
	}
	if publishErr := th.natsConnection.Publish(th.settings.NatsSubject, data); publishErr != nil {
		return publishErr
	}
	return nil
}

func (th *TelegramHandler) startCommand(update *Update) error {
	telegramUserID := strconv.FormatInt(update.Message.From.ID, 10)
	telegramUsername := update.Message.From.Username
	if insertErr := th.udao.Insert(telegramUserID, telegramUsername); insertErr != nil {
		return insertErr
	}
	notification := Notification{
		Type: startCommand,
		Text: welcomeText,
	}
	if sendNotificationErr := th.sendNotification(notification); sendNotificationErr != nil {
		return sendNotificationErr
	}
	return nil
}

func (th *TelegramHandler) animesCommand(update *Update) error {
	telegramUserID := strconv.FormatInt(update.Message.From.ID, 10)
	animes, animeErr := th.adao.ReadNotUserAnimes(telegramUserID)
	if animeErr != nil {
		return animeErr
	}
	notification := Notification{
		Type:   animesCommand,
		Text:   animesText,
		Animes: animes,
	}
	if sendNotificationErr := th.sendNotification(notification); sendNotificationErr != nil {
		return sendNotificationErr
	}
	return nil
}

func (th *TelegramHandler) subscriptionsCommand(update *Update) error {
	telegramUserID := strconv.FormatInt(update.Message.From.ID, 10)
	animes, animeErr := th.adao.ReadUserAnimes(telegramUserID)
	if animeErr != nil {
		return animeErr
	}
	notification := Notification{
		Type:   subscriptionsCommand,
		Text:   subscriptionsText,
		Animes: animes,
	}
	if sendNotificationErr := th.sendNotification(notification); sendNotificationErr != nil {
		return sendNotificationErr
	}
	return nil
}

func (th *TelegramHandler) defaultCommand(update *Update) error {
	telegramUserID := strconv.FormatInt(update.Message.From.ID, 10)
	userDTO, findUserErr := th.udao.Find(telegramUserID)
	if findUserErr != nil {
		return findUserErr
	}
	animeDTO, findAnimeErr := th.adao.Find(update.Message.Text)
	if findAnimeErr != nil {
		return findAnimeErr
	}
	if animeDTO == nil {
		return errors.New("Anime not found")
	}
	found, findErr := th.sdao.Find(userDTO.ID, animeDTO.ID)
	if findErr != nil {
		return findErr
	}
	notificationText := "Подписка "
	if found {
		if deleteErr := th.sdao.Delete(userDTO.ID, animeDTO.ID); deleteErr != nil {
			return deleteErr
		}
		notificationText += "удалена"
	} else {
		if insertErr := th.sdao.Insert(userDTO.ID, animeDTO.ID); insertErr != nil {
			return insertErr
		}
		notificationText += "добавлена"
	}
	notification := Notification{
		Type: defaultCommand,
		Text: notificationText,
	}
	if sendNotificationErr := th.sendNotification(notification); sendNotificationErr != nil {
		return sendNotificationErr
	}
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

//Notification struct
type Notification struct {
	Type   string         `json:"type"`
	Text   string         `json:"text"`
	Animes []dao.AnimeDTO `json:"animes"`
}
