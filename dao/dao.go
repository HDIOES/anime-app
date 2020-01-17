package dao

import (
	sql "database/sql"
	"log"
	"time"
)

//AnimeDAO struct
type AnimeDAO struct {
	Db *sql.DB
}

//AnimeDTO struct
type AnimeDTO struct {
	id            int64
	externalID    string
	rusName       string
	engName       string
	imageURL      string
	nextEpisodeAt time.Time
}

const userAnimesSQL = "SELECT (ANS.ID, ANS.EXTERNALID, ANS.RUSNAME, ANS.ENGNAME, ANS.IMAGEURL, ANS.NEXT_EPISODE_AT) FROM TELEGRAM_USERS TU" +
	" JOIN SUBCRIPTIONS SS ON (TU.USERNAME = $1 AND TU.ID = SS.TELEGRAM_USER_ID)" +
	" JOIN ANIMES ANS ON (ANS.ID = SS.ANIME_ID)"

const notUserAnimesSQL = "SELECT (ANS.ID, ANS.EXTERNALID, ANS.RUSNAME, ANS.ENGNAME, ANS.IMAGEURL, ANS.NEXT_EPISODE_AT) FROM ANIMES" +
	" EXCEPT " + userAnimesSQL

//ReadNotUserAnimes func
func (adao *AnimeDAO) ReadNotUserAnimes(externalID string) ([]AnimeDTO, error) {
	return adao.readAnimesBySQL(externalID, notUserAnimesSQL)
}

//ReadUserAnimes func
func (adao *AnimeDAO) ReadUserAnimes(externalID string) ([]AnimeDTO, error) {
	return adao.readAnimesBySQL(externalID, userAnimesSQL)
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
		var ID *sql.NullInt64
		var externalID *sql.NullString
		var rusname *sql.NullString
		var engname *sql.NullString
		var imageURL *sql.NullString
		var nextEpisodeAt *sql.NullString
		scanErr := result.Scan(&ID, &externalID, &rusname, &engname, &imageURL, &nextEpisodeAt)
		if scanErr != nil {
			result.Close()
			return nil, scanErr
		}
		animeDTO := AnimeDTO{}
		if ID.Valid {
			animeDTO.id = ID.Int64
		}
		if externalID.Valid {
			animeDTO.externalID = externalID.String
		}
		if rusname.Valid {
			animeDTO.rusName = rusname.String
		}
		if engname.Valid {
			animeDTO.engName = engname.String
		}
		if imageURL.Valid {
			animeDTO.imageURL = imageURL.String
		}
		if nextEpisodeAt.Valid {
			if time, parseErr := time.ParseInLocation("2016-06-22 19:10:25-07", nextEpisodeAt.String, time.Local); parseErr != nil {
				log.Println(parseErr)
			} else {
				animeDTO.nextEpisodeAt = time
			}
		}
	}
	return userAnimes, nil
}

//UserDAO struct
type UserDAO struct {
	Db *sql.DB
}

//UserDTO struct
type UserDTO struct {
	id               int64
	externalID       string
	telegramUsername string
}

//Insert func
func (udao *UserDAO) Insert(tx *sql.Tx, externalID string, username string) error {
	sqlStatement, stmtErr := tx.Prepare("INSERT INTO TELEGRAM_USERS (TELEGRAM_USER_ID, TELEGRAM_USERNAME) ($1, $2)")
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
func (sdao *SubscriptionDAO) Insert(tx *sql.Tx, userID int64, animeID int64) error {
	sqlStatement, stmtErr := tx.Prepare("INSERT INTO SUBSCRIPTIONS (TELEGRAM_USER_ID, ANIME_ID) ($1, $2)")
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

//Delete func
func (sdao *SubscriptionDAO) Delete(tx *sql.Tx, userID int64, animeID int64) error {
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
