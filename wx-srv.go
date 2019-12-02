package main

import (
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

const (
	TOKEN        = "lir419"
	LOG_FILENAME = "wx-srv.log"

	AppID     = "wxbf0b542fb9ba924d"
	AppSecret = "06d4d1888b7a320c9fad1c16eb9203f7"
)

var log = logrus.New()
var c = cache.New(5*time.Minute, 10*time.Minute)

func main() {
	//test()
	//return

	filename := LOG_FILENAME
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.WithError(err).Warnln("Failed to log to file, using default stderr")
	}
	log.Infoln("lir wx server start.")

	log.SetLevel(logrus.DebugLevel)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			CheckSign(w, r)
		case http.MethodPost:
			HandleMessage(w, r)
		default:
			log.Infof("unexpected req: %s\n", r.Method)
		}
	})

	log.Infof("access token: %s\n", GetAccessToken())

	http.ListenAndServe(":10419", nil)
}
