package dao

import (
	sql "database/sql"
	"time"
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

const (
	userAnimesSQL = "SELECT ANS.ID, ANS.EXTERNALID, ANS.RUSNAME, ANS.ENGNAME, ANS.IMAGEURL, ANS.NEXT_EPISODE_AT, ANS.NOTIFICATION_SENT FROM TELEGRAM_USERS AS TU" +
		" JOIN SUBSCRIPTIONS AS SS ON (TU.TELEGRAM_USER_ID = $1 AND TU.ID = SS.TELEGRAM_USER_ID)" +
		" JOIN ANIMES AS ANS ON (ANS.ID = SS.ANIME_ID)"

	notUserAnimesSQL = "SELECT ID, EXTERNALID, RUSNAME, ENGNAME, IMAGEURL, NEXT_EPISODE_AT, NOTIFICATION_SENT FROM ANIMES" +
		" EXCEPT " + userAnimesSQL

	findAnimeSQL        = "SELECT ID, EXTERNALID, RUSNAME, ENGNAME, IMAGEURL, NEXT_EPISODE_AT, NOTIFICATION_SENT FROM ANIMES WHERE ENGNAME = $1"
	findUserSQL         = "SELECT ID, TELEGRAM_USER_ID, TELEGRAM_USERNAME FROM TELEGRAM_USERS WHERE TELEGRAM_USER_ID = $1"
	findSubscriptionSQL = "SELECT TELEGRAM_USER_ID, ANIME_ID FROM SUBSCRIPTIONS WHERE TELEGRAM_USER_ID = $1 AND ANIME_ID = $2"
)

//Find func
func (adao *AnimeDAO) Find(engname string) (*AnimeDTO, error) {
	sqlStatement, stmtErr := adao.Db.Prepare(findAnimeSQL)
	if stmtErr != nil {
		return nil, stmtErr
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(engname)
	if resErr != nil {
		return nil, resErr
	}
	defer result.Close()
	if result.Next() {
		animeDTO, scanErr := adao.scanAsAnime(result)
		if scanErr != nil {
			return nil, scanErr
		}
		return animeDTO, nil
	}
	return nil, nil
}

//ReadNotUserAnimes func
func (adao *AnimeDAO) ReadNotUserAnimes(externalID string) ([]AnimeDTO, error) {
	return adao.readAnimesBySQL(externalID, notUserAnimesSQL)
}

//ReadUserAnimes func
func (adao *AnimeDAO) ReadUserAnimes(externalID string) ([]AnimeDTO, error) {
	return adao.readAnimesBySQL(externalID, userAnimesSQL)
}

func (adao *AnimeDAO) scanAsAnime(result *sql.Rows) (*AnimeDTO, error) {
	var ID *sql.NullInt64
	var externalID *sql.NullString
	var rusname *sql.NullString
	var engname *sql.NullString
	var imageURL *sql.NullString
	var nextEpisodeAt *PqTime
	var notificationSent *sql.NullBool
	scanErr := result.Scan(&ID, &externalID, &rusname, &engname, &imageURL, &nextEpisodeAt, &notificationSent)
	if scanErr != nil {
		return nil, scanErr
	}
	animeDTO := AnimeDTO{}
	if ID.Valid {
		animeDTO.ID = ID.Int64
	}
	if externalID.Valid {
		animeDTO.ExternalID = externalID.String
	}
	if rusname.Valid {
		animeDTO.RusName = rusname.String
	}
	if engname.Valid {
		animeDTO.EngName = engname.String
	}
	if imageURL.Valid {
		animeDTO.ImageURL = imageURL.String
	}
	if nextEpisodeAt.Valid {
		animeDTO.NextEpisodeAt = nextEpisodeAt.Time
	}
	if notificationSent.Valid {
		animeDTO.NotificationSent = notificationSent.Bool
	}
	return &animeDTO, nil
}

func (adao *AnimeDAO) readAnimesBySQL(externalID string, sqlStr string) ([]AnimeDTO, error) {
	sqlStatement, stmtErr := adao.Db.Prepare(sqlStr)
	if stmtErr != nil {
		return nil, stmtErr
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(externalID)
	if resErr != nil {
		return nil, resErr
	}
	userAnimes := make([]AnimeDTO, 0, 10)
	for result.Next() {
		animeDTO, scanErr := adao.scanAsAnime(result)
		if scanErr != nil {
			result.Close()
			return nil, scanErr
		}
		userAnimes = append(userAnimes, *animeDTO)
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
	sqlStatement, stmtErr := udao.Db.Prepare(findUserSQL)
	if stmtErr != nil {
		return nil, stmtErr
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(telegramID)
	if resErr != nil {
		return nil, resErr
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
		return nil, scanErr
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
func (udao *UserDAO) Insert(externalID string, username string) error {
	tx, txErr := udao.Db.Begin()
	if txErr != nil {
		return txErr
	}
	if insertErr := udao.insert(tx, externalID, username); insertErr != nil {
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

func (udao *UserDAO) insert(tx *sql.Tx, externalID string, username string) error {
	sqlStatement, stmtErr := tx.Prepare("INSERT INTO TELEGRAM_USERS (TELEGRAM_USER_ID, TELEGRAM_USERNAME) VALUES($1, $2)")
	if stmtErr != nil {
		return stmtErr
	}
	defer sqlStatement.Close()
	_, resErr := sqlStatement.Exec(externalID, username)
	if resErr != nil {
		return resErr
	}
	return nil
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

//Insert func
func (sdao *SubscriptionDAO) Insert(userID int64, animeID int64) error {
	tx, txErr := sdao.Db.Begin()
	if txErr != nil {
		return txErr
	}
	if insertErr := sdao.insert(tx, userID, animeID); insertErr != nil {
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

func (sdao *SubscriptionDAO) insert(tx *sql.Tx, userID int64, animeID int64) error {
	sqlStatement, stmtErr := tx.Prepare("INSERT INTO SUBSCRIPTIONS (TELEGRAM_USER_ID, ANIME_ID) VALUES($1, $2)")
	if stmtErr != nil {
		return stmtErr
	}
	defer sqlStatement.Close()
	_, resErr := sqlStatement.Exec(userID, animeID)
	if resErr != nil {
		return resErr
	}
	return nil
}

//Find func
func (sdao *SubscriptionDAO) Find(userID int64, animeID int64) (bool, error) {
	sqlStatement, stmtErr := sdao.Db.Prepare(findSubscriptionSQL)
	if stmtErr != nil {
		return false, stmtErr
	}
	defer sqlStatement.Close()
	result, resErr := sqlStatement.Query(userID, animeID)
	if resErr != nil {
		return false, resErr
	}
	defer result.Close()
	return result.Next(), nil
}

//Delete func
func (sdao *SubscriptionDAO) Delete(userID int64, animeID int64) error {
	tx, txErr := sdao.Db.Begin()
	if txErr != nil {
		return txErr
	}
	if insertErr := sdao.delete(tx, userID, animeID); insertErr != nil {
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

func (sdao *SubscriptionDAO) delete(tx *sql.Tx, userID int64, animeID int64) error {
	sqlStatement, stmtErr := tx.Prepare("DELETE FROM SUBSCRIPTIONS WHERE TELEGRAM_USER_ID = $1 AND ANIME_ID = $2")
	if stmtErr != nil {
		return stmtErr
	}
	defer sqlStatement.Close()
	_, resErr := sqlStatement.Exec(userID, animeID)
	if resErr != nil {
		return resErr
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
