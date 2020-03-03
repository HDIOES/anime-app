package dao

import (
	sql "database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

//AnimeDAO struct
type AnimeDAO struct {
	Db *sql.DB
}

//AnimeDTO struct
type AnimeDTO struct {
	ID               int64
	ExternalID       string
	RusName          string
	EngName          string
	ImageURL         string
	NextEpisodeAt    time.Time
	NotificationSent bool
}

//UserAnimeDTO struct
type UserAnimeDTO struct {
	AnimeDTO
	UserHasSubscription bool
}

const pageSize = 50

const (
	findAnimeByInternalIDAndByInternalUserIDSQL = "SELECT ANS.ID, ANS.EXTERNALID, ANS.RUSNAME, ANS.ENGNAME, ANS.IMAGEURL, ANS.NEXT_EPISODE_AT, ANS.NOTIFICATION_SENT, SS.ANIME_ID FROM ANIMES AS ANS" +
		" LEFT JOIN SUBSCRIPTIONS AS SS ON (ANS.ID = SS.ANIME_ID AND SS.TELEGRAM_USER_ID = $1) WHERE ANS.ID = $2"
	findAllAnimesBySentenceAndInternalUserIDSQL = "SELECT ANS.ID, ANS.EXTERNALID, ANS.RUSNAME, ANS.ENGNAME, ANS.IMAGEURL, ANS.NEXT_EPISODE_AT, ANS.NOTIFICATION_SENT, SS.ANIME_ID FROM ANIMES AS ANS" +
		" LEFT JOIN SUBSCRIPTIONS AS SS ON (ANS.ID = SS.ANIME_ID AND SS.TELEGRAM_USER_ID = $1) WHERE LOWER(ANS.ENGNAME) LIKE $2 OR LOWER(ANS.RUSNAME) LIKE $2"
	findUserByExternalIDSQL = "SELECT ID, TELEGRAM_USER_ID, TELEGRAM_USERNAME FROM TELEGRAM_USERS WHERE TELEGRAM_USER_ID = $1"
	findSubscriptionSQL     = "SELECT TELEGRAM_USER_ID, ANIME_ID FROM SUBSCRIPTIONS WHERE TELEGRAM_USER_ID = $1 AND ANIME_ID = $2"
	insertUserSQL           = "INSERT INTO TELEGRAM_USERS (TELEGRAM_USER_ID, TELEGRAM_USERNAME) VALUES($1, $2) RETURNING ID"
	insertSubscriptionSQL   = "INSERT INTO SUBSCRIPTIONS (TELEGRAM_USER_ID, ANIME_ID) VALUES($1, $2)"
	deleteSubscriptionSQL   = "DELETE FROM SUBSCRIPTIONS WHERE TELEGRAM_USER_ID = $1 AND ANIME_ID = $2"
)

//FindByUserIDAndInternalID func
func (adao *AnimeDAO) FindByUserIDAndInternalID(internalUserID, internalAnimeID int64) (*UserAnimeDTO, error) {
	sqlStatement, stmtErr := adao.Db.Prepare(findAnimeByInternalIDAndByInternalUserIDSQL)
	if stmtErr != nil {
		return nil, errors.WithStack(stmtErr)
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(internalUserID, internalAnimeID)
	if resErr != nil {
		return nil, errors.WithStack(resErr)
	}
	defer result.Close()
	if result.Next() {
		userAnimeDTO, scanErr := adao.scanAsUserAnime(result)
		if scanErr != nil {
			return nil, scanErr
		}
		return userAnimeDTO, nil
	}
	return nil, nil
}

//ReadUserAnimes func
func (adao *AnimeDAO) ReadUserAnimes(internalUserID int64, sentence string) ([]UserAnimeDTO, error) {
	return adao.readUserAnimesBySQL(internalUserID, sentence, findAllAnimesBySentenceAndInternalUserIDSQL)
}

func (adao *AnimeDAO) scanAsUserAnime(result *sql.Rows) (*UserAnimeDTO, error) {
	var ID *sql.NullInt64
	var externalID *sql.NullString
	var rusname *sql.NullString
	var engname *sql.NullString
	var imageURL *sql.NullString
	var nextEpisodeAt *PqTime
	var notificationSent *sql.NullBool
	var userID *sql.NullInt64
	scanErr := result.Scan(&ID, &externalID, &rusname, &engname, &imageURL, &nextEpisodeAt, &notificationSent, &userID)
	if scanErr != nil {
		return nil, errors.WithStack(scanErr)
	}
	userAnimeDTO := UserAnimeDTO{}
	if ID.Valid {
		userAnimeDTO.ID = ID.Int64
	}
	if externalID.Valid {
		userAnimeDTO.ExternalID = externalID.String
	}
	if rusname.Valid {
		userAnimeDTO.RusName = rusname.String
	}
	if engname.Valid {
		userAnimeDTO.EngName = engname.String
	}
	if imageURL.Valid {
		userAnimeDTO.ImageURL = imageURL.String
	}
	if nextEpisodeAt.Valid {
		userAnimeDTO.NextEpisodeAt = nextEpisodeAt.Time
	}
	if notificationSent.Valid {
		userAnimeDTO.NotificationSent = notificationSent.Bool
	}
	userAnimeDTO.UserHasSubscription = userID.Valid
	return &userAnimeDTO, nil
}

func (adao *AnimeDAO) readUserAnimesBySQL(internalUserID int64, sentence string, sqlStr string) ([]UserAnimeDTO, error) {
	sqlStatement, stmtErr := adao.Db.Prepare(sqlStr)
	if stmtErr != nil {
		return nil, errors.WithStack(stmtErr)
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(internalUserID, fmt.Sprintf("%%%s%%", sentence))
	if resErr != nil {
		return nil, errors.WithStack(resErr)
	}
	defer result.Close()
	userAnimes := make([]UserAnimeDTO, 0, pageSize)
	for result.Next() {
		userAnimeDTO, scanErr := adao.scanAsUserAnime(result)
		if scanErr != nil {
			return nil, scanErr
		}
		userAnimes = append(userAnimes, *userAnimeDTO)
	}
	return userAnimes, nil
}

//UserDAO struct
type UserDAO struct {
	Db *sql.DB
}

//UserDTO struct
type UserDTO struct {
	ID               int64
	ExternalID       string
	TelegramUsername string
}

//Find func
func (udao *UserDAO) Find(telegramID string) (*UserDTO, error) {
	sqlStatement, stmtErr := udao.Db.Prepare(findUserByExternalIDSQL)
	if stmtErr != nil {
		return nil, errors.WithStack(stmtErr)
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(telegramID)
	if resErr != nil {
		return nil, errors.WithStack(resErr)
	}
	defer result.Close()
	if result.Next() {
		userDTO, scanErr := udao.scanAsUser(result)
		if scanErr != nil {
			return nil, scanErr
		}
		return userDTO, nil
	}
	return nil, nil
}

func (udao *UserDAO) scanAsUser(result *sql.Rows) (*UserDTO, error) {
	var id *sql.NullInt64
	var telegramID *sql.NullString
	var telegramUsername *sql.NullString
	scanErr := result.Scan(&id, &telegramID, &telegramUsername)
	if scanErr != nil {
		return nil, errors.WithStack(scanErr)
	}
	userDTO := UserDTO{}
	if id.Valid {
		userDTO.ID = id.Int64
	}
	if telegramID.Valid {
		userDTO.ExternalID = telegramID.String
	}
	if telegramUsername.Valid {
		userDTO.TelegramUsername = telegramUsername.String
	}
	return &userDTO, nil
}

//Insert func
func (udao *UserDAO) Insert(externalID string, username string) (*UserDTO, error) {
	tx, txErr := udao.Db.Begin()
	if txErr != nil {
		return nil, errors.WithStack(txErr)
	}
	userDTO, insertErr := udao.insert(tx, externalID, username)
	if insertErr != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return nil, errors.WithStack(rollbackErr)
		}
		return nil, insertErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return nil, errors.WithStack(commitErr)
	}
	return userDTO, nil
}

func (udao *UserDAO) insert(tx *sql.Tx, externalID string, username string) (*UserDTO, error) {
	sqlStatement, stmtErr := tx.Prepare(insertUserSQL)
	if stmtErr != nil {
		return nil, errors.WithStack(stmtErr)
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(externalID, username)
	if resErr != nil {
		return nil, errors.WithStack(resErr)
	}
	defer result.Close()
	userDTO := UserDTO{
		ExternalID:       externalID,
		TelegramUsername: username,
	}
	if result.Next() {
		var ID *sql.NullInt64
		if err := result.Scan(&ID); err != nil {
			return nil, errors.WithStack(err)
		}
		if ID.Valid {
			userDTO.ID = ID.Int64
		}
	}
	return &userDTO, nil
}

//SubscriptionDAO struct
type SubscriptionDAO struct {
	Db *sql.DB
}

//SubcriptionDTO struct
type SubcriptionDTO struct {
	userID  int64
	animeID int64
}

//Find func
func (sdao *SubscriptionDAO) Find(userID int64, animeID int64) (bool, error) {
	sqlStatement, stmtErr := sdao.Db.Prepare(findSubscriptionSQL)
	if stmtErr != nil {
		return false, errors.WithStack(stmtErr)
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(userID, animeID)
	if resErr != nil {
		return false, errors.WithStack(resErr)
	}
	defer result.Close()
	return result.Next(), nil
}

//Insert func
func (sdao *SubscriptionDAO) Insert(userID int64, animeID int64) error {
	tx, txErr := sdao.Db.Begin()
	if txErr != nil {
		return errors.WithStack(txErr)
	}
	if insertErr := sdao.insert(tx, userID, animeID); insertErr != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.WithStack(rollbackErr)
		}
		return insertErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return errors.WithStack(commitErr)
	}
	return nil
}

func (sdao *SubscriptionDAO) insert(tx *sql.Tx, userID int64, animeID int64) error {
	sqlStatement, stmtErr := tx.Prepare(insertSubscriptionSQL)
	if stmtErr != nil {
		return errors.WithStack(stmtErr)
	}
	defer sqlStatement.Close()
	_, resErr := sqlStatement.Exec(userID, animeID)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	return nil
}

//Delete func
func (sdao *SubscriptionDAO) Delete(userID int64, animeID int64) error {
	tx, txErr := sdao.Db.Begin()
	if txErr != nil {
		return errors.WithStack(txErr)
	}
	if insertErr := sdao.delete(tx, userID, animeID); insertErr != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.WithStack(rollbackErr)
		}
		return insertErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return errors.WithStack(commitErr)
	}
	return nil
}

func (sdao *SubscriptionDAO) delete(tx *sql.Tx, userID int64, animeID int64) error {
	sqlStatement, stmtErr := tx.Prepare(deleteSubscriptionSQL)
	if stmtErr != nil {
		return errors.WithStack(stmtErr)
	}
	defer sqlStatement.Close()
	_, resErr := sqlStatement.Exec(userID, animeID)
	if resErr != nil {
		return errors.WithStack(resErr)
	}
	return nil
}

//PqTime struct
type PqTime struct {
	Time  time.Time
	Valid bool
}

//Scan func
func (pt *PqTime) Scan(value interface{}) error {
	if value == nil {
		pt.Valid = false
		return nil
	}
	pt.Time = value.(time.Time)
	pt.Valid = true
	return nil
}
