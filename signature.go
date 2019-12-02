package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"
)

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

type accessToken struct {
	Token  string `json:"access_token"`
	Expire int    `json:"expires_in"`
}

func GetAccessToken() string {
	token, found := c.Get("access_token")
	if found {
		return token.(string)
	}

	atRemote, err := getAccessTokenRemote()
	if err != nil {
		log.WithError(err).Warnln("Can't get access token")
		return ""
	}

	var at accessToken
	json.Unmarshal(atRemote, &at)
	c.Set("access_token", at.Token, time.Duration(at.Expire)*time.Second)
	return at.Token
}

func getAccessTokenRemote() ([]byte, error) {
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
