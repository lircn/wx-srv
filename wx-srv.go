package main

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const (
	TOKEN = "lir419"
)

func CheckSign(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "req: %s\n", r.URL.Path)
	signature := r.URL.Query().Get("signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	token := TOKEN
	tmpArr := []string{token, timestamp, nonce}
	sort.Strings(tmpArr)
	tmpStr := strings.Join(tmpArr, "")
	h := sha1.New()
	h.Write([]byte(tmpStr))
	tmpStr = string(h.Sum(nil))

	if tmpStr == signature {
		w.Write([]byte(r.URL.Query().Get("echostr")))
	} else {
		w.Write([]byte("error"))
	}
}

func main() {
	http.HandleFunc("/check_sign", CheckSign)
	http.ListenAndServe(":10419", nil)
}
