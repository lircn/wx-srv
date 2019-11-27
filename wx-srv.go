package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
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

func CheckSign(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	signature := r.URL.Query().Get("signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	token := TOKEN
	tmpArr := []string{token, timestamp, nonce}

	sort.Strings(tmpArr)
	tmpStr := strings.Join(tmpArr, "")

	h := sha1.New()
	h.Write([]byte(tmpStr))
	sha := hex.EncodeToString(h.Sum(nil))

	if sha == signature {
		w.Write([]byte(r.URL.Query().Get("echostr")))
	} else {
		w.Write([]byte("error"))
	}
}

type AccessToken struct {
	Token  string `json:"access_token"`
	Expire int    `json:"expires_in"`
}

func GetAccessToken() string {
	token, found := c.Get("access_token")
	if found {
		return token.(string)
	}

	atRemote, err := GetAccessTokenRemote()
	if err != nil {
		log.WithError(err).Warnln("Can't get access token")
		return ""
	}

	var at AccessToken
	json.Unmarshal(atRemote, &at)
	c.Set("access_token", at.Token, time.Duration(at.Expire)*time.Second)
	return at.Token
}

func GetAccessTokenRemote() ([]byte, error) {
	url := "https://api.weixin.qq.com/cgi-bin/token"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Add("grant_type", "client_credential")
	query.Add("appid", AppID)
	query.Add("secret", AppSecret)
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

type ReqMessage struct {
	CreateTime   int64  `xml:"CreateTime"`
	MsgId        int64  `xml:"MsgId"`
	URL          string `xml:"URL"`
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
}

type RepMessage struct {
	XMLName      xml.Name
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
}

func HandleMessage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Warnln("Can't read req body")
		return
	}

	reqMsg := ReqMessage{}
	xml.Unmarshal(data, &reqMsg)
	log.Debugf("req: %+v\n", reqMsg)

	repMsg := RepMessage{
		XMLName:      xml.Name{Local: "xml"},
		ToUserName:   reqMsg.FromUserName,
		FromUserName: reqMsg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      "测试",
	}
	repMsgData, err := xml.Marshal(repMsg)
	if err != nil {
		log.WithError(err).Warnln("Can't parse rep body")
		return
	}
	log.Debugf("rep: %+v\n", repMsg)

	w.Write(repMsgData)
}

func main() {

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

	http.ListenAndServe(":10419", nil)
}
