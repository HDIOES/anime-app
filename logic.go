package main

import (
	"net/http"

	"github.com/HDIOES/anime-app/dao"
)

//TelegramHandler struct
type TelegramHandler struct {
	udao *dao.UserDAO
	sdao *dao.SubscriptionDAO
	adao *dao.AnimeDAO
}

func (th *TelegramHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}
