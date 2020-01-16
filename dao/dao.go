package dao

import (
	"database/sql"
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

//ReadAll func
func (adao *AnimeDAO) ReadAll() ([]AnimeDTO, error) {
	return nil, nil
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
func (udao *UserDAO) Insert() error {
	return nil
}

//ReadAll func
func (udao *UserDAO) ReadAll() ([]UserDTO, error) {
	return nil, nil
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
func (sdao *SubscriptionDAO) Insert() error {
	return nil
}

//Delete func
func (sdao *SubscriptionDAO) Delete() error {
	return nil
}
