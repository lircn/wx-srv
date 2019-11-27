package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const (
	TOKEN = "lir419"
)

func CheckSign(w http.ResponseWriter, r *http.Request) {
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

func main() {
	http.HandleFunc("/", CheckSign)
	http.ListenAndServe(":10419", nil)
	fmt.Println("lir wx server start.")
}
