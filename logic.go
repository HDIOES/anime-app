package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"

	"github.com/HDIOES/anime-app/dao"
)

const (
	welcomeText        = "Данный бот предназначен для своевременного уведомления о выходе в эфир эпизодов ваших любимых аниме-сериалов"
	alertText          = "С возвращением! Ранее вы уже пользовались ботом, все ваши подписки сохранены"
	unknownCommandText = "Неизвестная команда"
)

const (
	startType       = "startType"
	answerQueryType = "answerQueryType"
	subscribeType   = "subscribeType"
	unsubscribeType = "unsubscribeType"
	defaultType     = "defaultType"
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
	reqReader, logReqErr := logRequest(r)
	if logReqErr != nil {
		HandleError(logReqErr)
		return
	}
	decoder := json.NewDecoder(reqReader)
	update := &Update{}
	decodeErr := decoder.Decode(update)
	if decodeErr != nil {
		HandleError(decodeErr)
		return
	}
	isMessage := update.Message != nil
	isInlineQuery := update.InlineQuery != nil
	isCallbackQuery := update.CallbackQuery != nil
	existedBefore := false
	var err error
	var userDTO *dao.UserDTO
	if isMessage {
		userDTO, existedBefore, err = th.checkAndSaveUserIfPossible(&update.Message.From)
	} else if isInlineQuery {
		userDTO, existedBefore, err = th.checkAndSaveUserIfPossible(&update.InlineQuery.From)
	} else if isCallbackQuery {
		userDTO, existedBefore, err = th.checkAndSaveUserIfPossible(&update.CallbackQuery.From)
	}
	if isMessage {
		if strings.HasPrefix(update.Message.Text, "/start") {
			parts := strings.SplitN(update.Message.Text, " ", 2)
			switch len(parts) {
			case 1:
				{
					err = th.startCommand(update.Message.From.ID, existedBefore)
				}
			case 2:
				{
					internalAnimeID, parseErr := strconv.ParseInt(parts[1], 10, 64)
					if parseErr != nil {
						err = parseErr
					} else {
						err = th.startCommandWithInternalAnimeID(update.Message.From.ID, userDTO.ID, internalAnimeID)
					}
				}
			default:
				{
					err = errors.New("Parse error")
				}
			}
		} else {
			err = errors.New("Parse error")
		}
	} else if isInlineQuery {
		err = th.inlineQueryCommand(userDTO.ID, update)
	} else if isCallbackQuery {
		parts := strings.SplitN(update.CallbackQuery.Data, " ", 2)
		if len(parts) == 2 && update.CallbackQuery.Message != nil {
			command := parts[0]
			internalAnimeID, parseErr := strconv.ParseInt(parts[1], 10, 64)
			if parseErr != nil {
				err = errors.WithStack(parseErr)
			}
			switch command {
			case "sub":
				{
					err = th.subscribeCommand(
						userDTO.ID,
						internalAnimeID,
						update.CallbackQuery.Message.Chat.ID,
						update.CallbackQuery.Message.MessageID)
				}
			case "unsub":
				{
					err = th.unsubscribeCommand(
						userDTO.ID,
						internalAnimeID,
						update.CallbackQuery.Message.Chat.ID,
						update.CallbackQuery.Message.MessageID)
				}
			default:
				{
					err = errors.New("Parse error")
				}
			}
		} else {
			err = errors.New("Parse error")
		}
	}
	if err != nil {
		HandleError(err)
	}
}

func (th *TelegramHandler) checkAndSaveUserIfPossible(user *User) (userDTO *dao.UserDTO, existedBefore bool, err error) {
	telegramUserID := strconv.FormatInt(user.ID, 10)
	telegramUsername := user.Username
	userDto, findErr := th.udao.Find(telegramUserID)
	if findErr != nil {
		return nil, false, findErr
	}
	if userDto == nil {
		if newUserDTO, insertErr := th.udao.Insert(telegramUserID, telegramUsername); insertErr != nil {
			return newUserDTO, false, insertErr
		}
		return nil, false, nil
	}
	return userDto, true, nil
}

func (th *TelegramHandler) sendNtsMessage(ntsMessage *TelegramCommandMessage) error {
	data, dataErr := json.Marshal(ntsMessage)
	if dataErr != nil {
		return errors.WithStack(dataErr)
	}
	if publishErr := th.natsConnection.Publish(th.settings.NatsSubject, data); publishErr != nil {
		return errors.WithStack(publishErr)
	}
	return nil
}

func (th *TelegramHandler) startCommand(userTelegramID int64, existedBefore bool) error {
	ntsMessage := TelegramCommandMessage{
		TelegramID: userTelegramID,
		Type:       startType,
	}
	if !existedBefore {
		ntsMessage.Text = welcomeText
	} else {
		ntsMessage.Text = alertText
	}
	if err := th.sendNtsMessage(&ntsMessage); err != nil {
		return err
	}
	return nil
}

func (th *TelegramHandler) startCommandWithInternalAnimeID(userTelegramID, internalUserID, internalAnimeID int64) error {
	ntsMessage := TelegramCommandMessage{
		TelegramID: userTelegramID,
		Type:       startType,
	}
	userAnimeDto, err := th.adao.FindByUserIDAndInternalID(internalUserID, internalAnimeID)
	if err != nil {
		return err
	}
	ntsMessage.InlineAnime = &InlineAnime{
		InternalID:           internalAnimeID,
		AnimeName:            userAnimeDto.EngName,
		AnimeThumbnailPicURL: userAnimeDto.ImageURL,
		UserHasSubscription:  userAnimeDto.UserHasSubscription,
	}
	if err := th.sendNtsMessage(&ntsMessage); err != nil {
		return err
	}
	return nil
}

func (th *TelegramHandler) inlineQueryCommand(internalUserID int64, update *Update) error {
	userAnimes, err := th.adao.ReadUserAnimes(internalUserID, update.InlineQuery.Query)
	if err != nil {
		return err
	}
	ntsMessage := TelegramCommandMessage{
		Type:          answerQueryType,
		InlineQueryID: update.InlineQuery.ID,
	}
	ntsMessage.InlineAnimes = make([]InlineAnime, len(userAnimes))
	for _, userAnime := range userAnimes {
		ntsMessage.InlineAnimes = append(ntsMessage.InlineAnimes, InlineAnime{
			InternalID:           userAnime.ID,
			AnimeName:            userAnime.EngName,
			AnimeThumbnailPicURL: th.settings.ShikimoriURL + userAnime.ImageURL,
			UserHasSubscription:  userAnime.UserHasSubscription,
		})
	}
	if err := th.sendNtsMessage(&ntsMessage); err != nil {
		return err
	}
	return nil
}

func (th *TelegramHandler) subscribeCommand(internalUserID, internalAnimeID int64, chatID, messageID int64) error {
	found, err := th.sdao.Find(internalUserID, internalAnimeID)
	if err != nil {
		return err
	}
	if found {
		if err := th.defaultCommand(internalUserID); err != nil {
			return err
		}
	} else {
		if err := th.sdao.Insert(internalUserID, internalAnimeID); err != nil {
			return err
		}
		ntsMessage := TelegramCommandMessage{
			Type:            subscribeType,
			ChatID:          chatID,
			MessageID:       messageID,
			InternalAnimeID: internalAnimeID,
		}
		if err := th.sendNtsMessage(&ntsMessage); err != nil {
			return err
		}
	}
	return nil
}

func (th *TelegramHandler) unsubscribeCommand(internalUserID, internalAnimeID, chatID, messageID int64) error {
	found, err := th.sdao.Find(internalUserID, internalAnimeID)
	if err != nil {
		return err
	}
	if found {
		if err := th.sdao.Delete(internalUserID, internalAnimeID); err != nil {
			return err
		}
		ntsMessage := TelegramCommandMessage{
			Type:            unsubscribeType,
			ChatID:          chatID,
			MessageID:       messageID,
			InternalAnimeID: internalAnimeID,
		}
		if err := th.sendNtsMessage(&ntsMessage); err != nil {
			return err
		}
	} else {
		if err := th.defaultCommand(internalAnimeID); err != nil {
			return err
		}
	}
	return nil
}

func (th *TelegramHandler) defaultCommand(userTelegramID int64) error {
	nstMessage := TelegramCommandMessage{
		TelegramID: userTelegramID,
		Type:       defaultType,
		Text:       unknownCommandText,
	}
	if sendNstMessageErr := th.sendNtsMessage(&nstMessage); sendNstMessageErr != nil {
		return sendNstMessageErr
	}
	return nil
}

//Update struct
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	InlineQuery   *InlineQuery   `json:"inline_query"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

//Message struct
type Message struct {
	MessageID int64  `json:"message_id"`
	From      User   `json:"from"`
	Text      string `json:"text"`
	Chat      Chat   `json:"chat"`
}

//InlineQuery struct
type InlineQuery struct {
	ID     string `json:"id"`
	From   User   `json:"from"`
	Query  string `json:"query"`
	Offset string `json:"offset"`
}

//CallbackQuery struct
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Data    string   `json:"data"`
	Message *Message `json:"message"`
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

//Chat struct
type Chat struct {
	ID int64 `json:"id"`
}

//TelegramCommandMessage struct
type TelegramCommandMessage struct {
	Type string `json:"type"`
	//fields for notification and /start
	TelegramID  int64        `json:"telegramId"`
	Text        string       `json:"text"`
	InlineAnime *InlineAnime `json:"inlineAnime"`
	//inline query fields
	InlineQueryID string        `json:"inlineQueryId"`
	InlineAnimes  []InlineAnime `json:"inlineAnimes"`
	//fields for subscribe/unsubscribe action
	ChatID          int64  `json:"chatId"`
	MessageID       int64  `json:"messageId"`
	CallbackQueryID string `json:"callback_query_id"`
	InternalAnimeID int64  `json:"internal_anime_id"`
}

//InlineAnime struct
type InlineAnime struct {
	InternalID           int64  `json:"id"`
	AnimeName            string `json:"animeName"`
	AnimeThumbnailPicURL string `json:"animeThumbNailPicUrl"`
	UserHasSubscription  bool   `json:"userHasSubscription"`
}
