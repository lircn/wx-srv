package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type reqMessage struct {
	CreateTime   int64  `xml:"CreateTime"`
	MsgId        int64  `xml:"MsgId"`
	URL          string `xml:"URL"`
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	Event        string `xml:"Event"`
}

type articleContent struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"Title"`
	Description string   `xml:"Description"`
	PicUrl      string   `xml:""`
	Url         string   `xml:"Url"`
}

type repMessage struct {
	XMLName      xml.Name         `xml:"xml"`
	ToUserName   string           `xml:"ToUserName"`
	FromUserName string           `xml:"FromUserName"`
	CreateTime   int64            `xml:"CreateTime"`
	MsgType      string           `xml:"MsgType"`
	Content      string           `xml:"Content"`
	ArticleCount int              `xml:"ArticleCount"`
	Articles     []articleContent `xml:"Articles>item"`
}

const (
	/* 腾讯AI智能闲聊 */
	AIChatAppId  = "2125052580"
	AIChatAppKey = "p67sMgph5IUiomcH"

	HeadAgent = "Mozilla/5.0 (Windows NT 10.0; WOW64; rv:60.0) Gecko/20100101 Firefox/60.0"

	ApiHttpTimeout = time.Second * 3
)

var (
	msgSubscribe   = "欢迎光临！\n回复“帮助”查看帮助信息"
	msgScan        = "欢迎再次光临！\n是在哪里看到我的呢？"
	msgUnsubscribe = "再见~ \n欢迎下次再来！"

	msgHelp = `1. 回复“手气”测试手气
2. 回复“主页”获取主页链接
3. 回复“彩虹屁”挨夸
4. 回复“土味”享受惊喜
5. 回复“垃圾 xx”查询xx是什么垃圾类型
6. 或者直接聊天~`
	msgHome = "http://blog.zaynli.com"
)

func HandleMessage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Warnln("Can't read req body")
		return
	}

	reqMsg := reqMessage{}
	xml.Unmarshal(data, &reqMsg)
	log.Debugf("raw req: %s\n", data)
	log.Debugf("req: %+v\n", reqMsg)

	repData := []byte("")
	switch reqMsg.MsgType {
	case "text":
		repData = handleMsgText(&reqMsg)
	case "event":
		repData = handleMsgEvent(&reqMsg)
	default:
	}

	log.Debugf("write: %s\n", string(repData))
	w.Write(repData)
}

func handleMsgEvent(reqMsg *reqMessage) []byte {
	repData := []byte("")
	switch reqMsg.Event {
	case "subscribe":
		repData = makeMsgText(reqMsg, msgSubscribe)
	case "SCAN":
		repData = makeMsgText(reqMsg, msgScan)
	case "unsubscribe":
		repData = makeMsgText(reqMsg, msgUnsubscribe)
	default:
	}
	return repData
}

var pokerTypeMap = map[int32]string{
	0: "黑桃",
	1: "红桃",
	2: "方片",
	3: "梅花",
}

var pokerNumMap = map[int32]string{
	0: "A", 1: "1", 2: "2", 3: "3", 4: "4",
	5: "5", 6: "6", 7: "7", 8: "8", 9: "9",
	10: "10", 11: "J", 12: "Q", 13: "K",
}

var coinMap = map[int32]string{
	0: "正面",
	1: "反面",
}

var rspMap = map[int32]string{
	0: "石头",
	1: "剪刀",
	2: "布",
}

var slotMachMap = map[int32]string{
	0: "⑦",
	1: "💎",
	2: "📚",
	3: "🍉",
	4: "🐒",
	5: "🐑",
	6: "🍑",
	7: "🚗",
}

var tuweiList []string

func init() {
	rand.Seed(time.Now().UnixNano())

	file, err := os.Open("tuwei.db")
	if err != nil {
		log.WithError(err).Warnln("Failed to open tuwei")
		return
	}
	defer file.Close()

	br := bufio.NewReader(file)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		tuweiList = append(tuweiList, string(a))
	}
}

func handleMsgTextLucky(reqMsg *reqMessage) []byte {
	var b bytes.Buffer

	coin := coinMap[rand.Int31n(2)]
	fmt.Fprintf(&b, "硬币：%s\n", coin)

	// Rock-Scissors-Paper
	rsp := rspMap[rand.Int31n(3)]
	fmt.Fprintf(&b, "猜拳：%s\n", rsp)

	dice := rand.Int31n(6)
	fmt.Fprintf(&b, "骰子：%d点\n", dice+1)

	pokerType := pokerTypeMap[rand.Int31n(4)]
	pokerNum := pokerNumMap[rand.Int31n(13)]
	fmt.Fprintf(&b, "扑克：%s%s\n", pokerType, pokerNum)

	// Slot machine
	slotMach := slotMachMap[rand.Int31n(7)]
	slotMach += slotMachMap[rand.Int31n(7)]
	slotMach += slotMachMap[rand.Int31n(7)]
	fmt.Fprintf(&b, "老虎机：%s\n", slotMach)

	return makeMsgText(reqMsg, b.String())
}

func aiGetReqSign(params map[string]string) string {
	str := ""
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		str += k + "=" + url.QueryEscape(params[k]) + "&"
	}
	str += "app_key=" + AIChatAppKey
	return strings.ToUpper(GetMD5Hash(str))
}

func aiChat(session string, in string) string {
	errorStr := "突然有点不想聊天"
	params := map[string]string{
		"app_id":     AIChatAppId,
		"session":    session,
		"question":   in,
		"time_stamp": strconv.FormatInt(time.Now().Unix(), 10),
		"nonce_str":  RandString(16),
	}
	params["sign"] = aiGetReqSign(params)

	URL := "https://api.ai.qq.com/fcgi-bin/nlp/nlp_textchat"
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		log.WithError(err).Warnln("Can't make ai req")
		return errorStr
	}

	query := req.URL.Query()
	for k, v := range params {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()

	var client = &http.Client{
		Timeout: ApiHttpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warnln("Can't get ai api")
		return errorStr
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Warnln("Can't get ai api")
		return errorStr
	}

	type aiRepData struct {
		Session string `json:"session"`
		Answer  string `json:"answer"`
	}
	type aiRepMsg struct {
		Ret  int       `json:"ret"`
		Msg  string    `json:"msg"`
		Data aiRepData `json:"data"`
	}
	var arm aiRepMsg
	json.Unmarshal(body, &arm)
	if arm.Ret != 0 {
		log.Warn("Ai return %s", arm.Msg)
		return errorStr
	}
	return arm.Data.Answer
}

func handleMsgTextNormal(reqMsg *reqMessage) []byte {
	log.Debugf("normal in: %s\n", reqMsg.Content)
	str := aiChat(reqMsg.FromUserName, reqMsg.Content)
	if str == "" {
		str = "请讲普通话"
	}
	log.Debugf("normal out: %s\n", str)
	return makeMsgText(reqMsg, str)
}

func caiHongPi() string {
	errorStr := "感觉今天不适合彩虹屁"
	var client = &http.Client{
		Timeout: ApiHttpTimeout,
	}
	resp, err := client.Get("https://chp.shadiao.app/api.php")
	if err != nil {
		log.WithError(err).Warnln("Can't get caihongpi")
		return errorStr
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Warnln("Can't read caihongpi")
		return errorStr
	}
	return string(body)
}

func lookupRubbish(key string) string {
	errorStr := "暂时不认识这个垃圾哦"
	key = strings.TrimSpace(key)

	URL := "http://www.atoolbox.net/api/GetRefuseClassification.php"
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		log.WithError(err).Warnln("Can't make rubbish req")
		return errorStr
	}

	query := req.URL.Query()
	query.Add("key", key)
	req.URL.RawQuery = query.Encode()
	req.Header.Add("User-Agent", HeadAgent)

	var client = &http.Client{
		Timeout: ApiHttpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warnln("Can't get rubbish")
		return errorStr
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Warnln("Can't get rubbish resp")
		return errorStr
	}
	if len(body) < 12 {
		return "不认识这个哦"
	}
	type rubbishItem struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	var rubbishMsg map[string]rubbishItem
	json.Unmarshal(body, &rubbishMsg)

	result := make(map[string][]string)
	for _, v := range rubbishMsg {
		result[v.Type] = append(result[v.Type], v.Name)
	}

	str := ""
	for k, v := range result {
		str += "\n" + k + ":\n"
		for _, item := range v {
			str += item + " "
		}
	}
	return str
}

func handleMsgText(reqMsg *reqMessage) []byte {
	ctx := reqMsg.Content

	if len(ctx) > 10 && ctx[:7] == "垃圾 " {
		return makeMsgText(reqMsg, lookupRubbish(ctx[7:]))
	}

	switch ctx {
	case "帮助":
		fallthrough
	case "help":
		return makeMsgText(reqMsg, msgHelp)

	case "主页":
		fallthrough
	case "home":
		return makeMsgText(reqMsg, msgHome)

	case "手气":
		fallthrough
	case "lucky":
		return handleMsgTextLucky(reqMsg)

	case "彩虹屁":
		return makeMsgText(reqMsg, caiHongPi())

	case "土味":
		return handleMsgTextTuwei(reqMsg)

	default:
		return handleMsgTextNormal(reqMsg)
	}
}

var (
	tuweiUrls = []string{
		"",
		"https://api.lovelive.tools/api/SweetNothings",
		"http://v1.alapi.cn/api/soul?format=text",
		"http://v1.alapi.cn/api/qinghua?type=text",
	}
)

func tuweiLocal() string {
	return tuweiList[rand.Intn(len(tuweiList))]
}

func handleMsgTextTuwei(reqMsg *reqMessage) []byte {
	idx := rand.Intn(len(tuweiUrls))
	log.Debugf("make tuwei idx %d", idx)
	if idx == 0 {
		return makeMsgText(reqMsg, tuweiLocal())
	}

	url := tuweiUrls[idx]
	var client = &http.Client{
		Timeout: ApiHttpTimeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		log.WithError(err).Warnln("Can't get %s", url)
		return makeMsgText(reqMsg, tuweiLocal())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Warnln("Can't read caihongpi")
		return makeMsgText(reqMsg, tuweiLocal())
	}
	return makeMsgText(reqMsg, string(body))
}

func makeMsgText(reqMsg *reqMessage, content string) []byte {
	repMsg := repMessage{
		ToUserName:   reqMsg.FromUserName,
		FromUserName: reqMsg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      content,
	}
	repMsgData, err := xml.Marshal(repMsg)
	if err != nil {
		log.WithError(err).Warnln("Can't make msg text")
		return nil
	}
	return repMsgData
}

func makeMsgImage(reqMsg *reqMessage, content string) []byte {
	repMsg := repMessage{
		ToUserName:   reqMsg.FromUserName,
		FromUserName: reqMsg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "image",
		Content:      content,
	}
	repMsgData, err := xml.Marshal(repMsg)
	if err != nil {
		log.WithError(err).Warnln("Can't make msg text")
		return nil
	}
	return repMsgData
}

func makeMsgArticle(reqMsg *reqMessage, title, desc, picUrl, url string) []byte {
	article := articleContent{
		Title:       "title",
		Description: "Description",
		PicUrl:      picUrl,
		Url:         url,
	}
	repMsg := repMessage{
		ToUserName:   reqMsg.FromUserName,
		FromUserName: reqMsg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "news",
		ArticleCount: 1,
		Articles:     []articleContent{article},
	}
	repMsgData, err := xml.Marshal(repMsg)
	if err != nil {
		log.WithError(err).Warnln("Can't make msg text")
		return nil
	}
	return repMsgData
}

func test() {
	ctx := "帮助 哈哈"
	if ctx[:7] == "帮助 " {
		fmt.Println("test")
		fmt.Println(ctx[6:])

	}
	fmt.Println("test2")
}
