package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"

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
	natsConnection *nats.Conn
	settings       *Settings
}

func (th *TelegramHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	update := &Update{}
	decodeErr := decoder.Decode(update)
	if decodeErr != nil {
		HandleError(decodeErr)
		th.setStatus(w, 400)
	}
	switch update.Message.Text {
	case "/start":
		if err := th.startCommand(update); err != nil {
			HandleError(err)
			th.setStatus(w, 400)
		}
	case "/animes":
		if err := th.animesCommand(update); err != nil {
			HandleError(err)
			th.setStatus(w, 400)
		}
	case "/subscriptions":
		if err := th.subscriptionsCommand(update); err != nil {
			HandleError(err)
			th.setStatus(w, 400)
		}
	default:
		if err := th.defaultCommand(update); err != nil {
			HandleError(err)
			th.setStatus(w, 400)
		}
	}
}

func (th *TelegramHandler) setStatus(w http.ResponseWriter, httpStatus int) {
	w.WriteHeader(httpStatus)
}

func (th *TelegramHandler) sendNotification(notification Notification) error {
	data, dataErr := json.Marshal(notification)
	if dataErr != nil {
		return errors.WithStack(dataErr)
	}
	if publishErr := th.natsConnection.Publish(th.settings.NatsSubject, data); publishErr != nil {
		return errors.WithStack(publishErr)
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
		TelegramID: update.Message.From.ID,
		Type:       startCommand,
		Text:       welcomeText,
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
	animeNames := make([]string, 0, len(animes))
	for _, anime := range animes {
		animeNames = append(animeNames, anime.EngName)
	}
	notification := Notification{
		TelegramID: update.Message.From.ID,
		Type:       animesCommand,
		Text:       animesText,
		Animes:     animeNames,
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
	animeNames := make([]string, 0, len(animes))
	for _, anime := range animes {
		animeNames = append(animeNames, anime.EngName)
	}
	notification := Notification{
		TelegramID: update.Message.From.ID,
		Type:       subscriptionsCommand,
		Text:       subscriptionsText,
		Animes:     animeNames,
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
		TelegramID: update.Message.From.ID,
		Type:       defaultCommand,
		Text:       notificationText,
	}
	if sendNotificationErr := th.sendNotification(notification); sendNotificationErr != nil {
		return sendNotificationErr
	}
	return nil
}

//Update struct
type Update struct {
	UpdateID int64   `json:"update_id"`
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
	TelegramID int64    `json:"telegramId"`
	Type       string   `json:"type"`
	Text       string   `json:"text"`
	Animes     []string `json:"animes"`
	WebhookURL string   `json:"webhookUrl"`
}
