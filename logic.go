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
	welcomeText           = "Данный бот предназначен для своевременного уведомления о выходе в эфир эпизодов ваших любимых аниме-сериалов"
	alertText             = "С возвращением! Ранее вы уже пользовались ботом, все ваши подписки сохранены"
	unknownCommandText    = "Неизвестная команда"
	subscribeButtonText   = "Подписаться"
	unsubscribeButtonText = "Отписаться"
	redirectButtonText    = "Подробности"
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
	}
	decoder := json.NewDecoder(reqReader)
	update := &Update{}
	decodeErr := decoder.Decode(update)
	if decodeErr != nil {
		HandleError(decodeErr)
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
	if err != nil {
		HandleError(err)
	}
	if isMessage {
		parts := strings.SplitN(update.Message.Text, " ", 2)
		command := parts[0]
		switch command {
		case "/start":
			countOfSubstrings := len(parts)
			if countOfSubstrings == 1 {
				if err := th.startCommand(update.Message.From.ID, existedBefore); err != nil {
					HandleError(err)
				}
			} else {
				internalAnimeID, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					HandleError(errors.WithStack(err))
				}
				if err := th.startCommandWithArgument(update.Message.From.ID, internalAnimeID, existedBefore); err != nil {
					HandleError(err)
				}
			}
		default:
			if err := th.defaultCommand(update.Message.From.ID); err != nil {
				HandleError(err)
			}
		}
	} else if isInlineQuery {
		if err := th.inlineQueryCommand(userDTO.ID, update); err != nil {
			HandleError(err)
		}
	} else if isCallbackQuery {
		parts := strings.SplitN(update.CallbackQuery.Data, " ", 2)
		countOfSubstrings := len(parts)
		if countOfSubstrings == 2 {
			command := parts[0]
			internalAnimeID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				HandleError(errors.WithStack(err))
			}
			switch command {
			case "sub":
				{
					if err := th.subscribeCommand(userDTO.ID, internalAnimeID); err != nil {
						HandleError(err)
					}
				}
			case "unsub":
				{
					if err := th.unsubscribeCommand(userDTO.ID, internalAnimeID); err != nil {
						HandleError(err)
					}
				}
			}
		}
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

func (th *TelegramHandler) startCommandWithArgument(userTelegramID, internalAnimeID int64, existedBefore bool) error {
	userAnimeDTO, err := th.adao.FindByUserIDAndInternalID(userTelegramID, internalAnimeID)
	if err != nil {
		return err
	}
	if userAnimeDTO != nil {
		//StartCommandWithoutArgs
		ntsMessage := TelegramCommandMessage{
			TelegramID: userTelegramID,
			AnimeInfo: InlineAnime{
				InternalID:           userAnimeDTO.ID,
				AnimeName:            userAnimeDTO.EngName,
				AnimeThumbnailPicURL: userAnimeDTO.ImageURL,
			},
		}
		if userAnimeDTO.UserHasSubscription {
			ntsMessage.Type = unsubscribeType
			ntsMessage.AnimeInfo.BottomInlineButton = subscribeButtonText
		} else {
			ntsMessage.Type = subscribeType
			ntsMessage.AnimeInfo.BottomInlineButton = unsubscribeButtonText
		}
		if err := th.sendNtsMessage(&ntsMessage); err != nil {
			return err
		}
	} else {
		//DefaultCommand
		if err := th.defaultCommand(userTelegramID); err != nil {
			return err
		}
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

func (th *TelegramHandler) inlineQueryCommand(internalUserID int64, update *Update) error {
	ntsMessage := TelegramCommandMessage{
		Type:          answerQueryType,
		InlineQueryID: update.InlineQuery.ID,
	}
	userAnimes, err := th.adao.ReadUserAnimes(internalUserID, update.InlineQuery.Query)
	if err != nil {
		return err
	}
	countOfUserAnimes := len(userAnimes)
	ntsMessage.InlineAnimes = make([]InlineAnime, countOfUserAnimes)
	for _, userAnime := range userAnimes {
		ntsMessage.InlineAnimes = append(ntsMessage.InlineAnimes, InlineAnime{
			InternalID:           userAnime.ID,
			AnimeName:            userAnime.EngName,
			AnimeThumbnailPicURL: userAnime.ImageURL,
			BottomInlineButton:   redirectButtonText,
			UserHasSubscription:  userAnime.UserHasSubscription,
		})
	}
	if err := th.sendNtsMessage(&ntsMessage); err != nil {
		return err
	}
	return nil
}

func (th *TelegramHandler) subscribeCommand(userID, animeID int64) error {
	found, err := th.sdao.Find(userID, animeID)
	if err != nil {
		return err
	}
	if found {
		if err := th.defaultCommand(userID); err != nil {
			return err
		}
	} else {
		if err := th.sdao.Insert(userID, animeID); err != nil {
			return err
		}
	}
	return nil
}

func (th *TelegramHandler) unsubscribeCommand(userID, animeID int64) error {
	found, err := th.sdao.Find(userID, animeID)
	if err != nil {
		return err
	}
	if found {
		if err := th.sdao.Delete(userID, animeID); err != nil {
			return err
		}
	} else {
		if err := th.defaultCommand(userID); err != nil {
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
	ID   string `json:"id"`
	From User   `json:"from"`
	Data string `json:"data"`
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

//TelegramCommandMessage struct
type TelegramCommandMessage struct {
	Type string `json:"type"`
	//fields for notification and /start without arguments
	TelegramID int64  `json:"telegramId"`
	Text       string `json:"text"`
	//inline query fields
	InlineQueryID string        `json:"inlineQueryId"`
	InlineAnimes  []InlineAnime `json:"inlineAnimes"`
	//fields for anime information after typing '/start shikimoriId' in private chat
	AnimeInfo InlineAnime `json:"animeInfo"`
}

//InlineAnime struct
type InlineAnime struct {
	InternalID           int64  `json:"id"`
	AnimeName            string `json:"animeName"`
	AnimeThumbnailPicURL string `json:"animeThumbNailPicUrl"`
	UserHasSubscription  bool   `json:"userHasSubscription"`
	BottomInlineButton   string `json:"bottomInlineButton"`
}
