package jd_cookie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/cdle/sillyGirl/core"
	"github.com/gorilla/websocket"
)

var jd_cookie = core.NewBucket("jd_cookie")

var mhome sync.Map

func init() {
	go RunServer()
	core.AddCommand("", []core.Function{
		{
			Rules: []string{`raw ^登录$`, `raw ^登陆$`, `raw ^h$`},
			Handle: func(s core.Sender) interface{} {
				if groupCode := jd_cookie.Get("groupCode"); !s.IsAdmin() && groupCode != "" && s.GetChatID() != 0 && !strings.Contains(groupCode, fmt.Sprint(s.GetChatID())) {
					return nil
				}
				if c == nil || s.GetImType() == "wxmp" {
					tip := jd_cookie.Get("tip")
					if tip == "" {
						if s.IsAdmin() {
							return jd_cookie.Get("tip", "阿东不行啦，更改登录提示指令，set jd_cookie tip ?")
						} else {
							tip = "暂时无法使用短信登录。"
						}
					}

					return tip
				}
				uid := time.Now().UnixNano()
				cry := make(chan string, 1)
				mhome.Store(uid, cry)
				stop := false
				var cookie *string
				sendMsg := func(msg string) {
					c.WriteJSON(map[string]interface{}{
						"time":         time.Now().Unix(),
						"self_id":      jd_cookie.GetInt("selfQid"),
						"post_type":    "message",
						"message_type": "private",
						"sub_type":     "friend",
						"message_id":   time.Now().UnixNano(),
						"user_id":      uid,
						"message":      msg,
						"raw_message":  msg,
						"font":         456,
						"sender": map[string]interface{}{
							"nickname": "傻妞",
							"sex":      "female",
							"age":      18,
						},
					})
				}
				defer func() {
					cry <- "stop"
					mhome.Delete(uid)
					if cookie != nil {
						s.SetContent(*cookie)
						core.Senders <- s
					}
					sendMsg("q")
				}()
				go func() {
					for {
						msg := <-cry
						if msg == "stop" {
							break
						}
						msg = strings.Replace(msg, "登陆", "登录", -1)
						if strings.Contains(msg, "不占资源") {
							msg += "\n" + "4.取消"
						}
						if strings.Contains(msg, "青龙状态") {
							sendMsg("1")
							continue
						}
						if strings.Contains(msg, "pt_key") {
							cookie = &msg
							stop = true
							s.SetContent("q")
							core.Senders <- s
						}
						if cookie == nil {
							if strings.Contains(msg, "已点击登录") {
								continue
							}

							s.Reply(regexp.MustCompile(".*").FindString(msg))
						}
					}
				}()
				sendMsg("h")
				for {
					if stop == true {
						break
					}
					s.Await(s, func(s core.Sender) interface{} {
						msg := s.GetContent()
						if msg == "q" || msg == "exit" || msg == "退出" || msg == "10" || msg == "4" {
							stop = true
							if cookie == nil {
								s.Reply("取消登录")
							} else {
								s.Reply("登录成功")
							}
						}
						sendMsg(s.GetContent())
						return nil
					}, `[\s\S]+`)
				}
				return nil
			},
		},
	})
	// if jd_cookie.GetBool("enable_aaron", false) {
	core.Senders <- &core.Faker{
		Message: "ql cron disable https://github.com/Aaron-lv/sync.git",
	}
	core.Senders <- &core.Faker{
		Message: "ql cron disable task Aaron-lv_sync_jd_scripts_jd_city.js",
	}
	// }
}

var c *websocket.Conn

func RunServer() {
	addr := jd_cookie.Get("adong_addr")
	if addr == "" {
		return
	}
	defer func() {
		time.Sleep(time.Second * 2)
		RunServer()
	}()
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws/event"}
	logs.Info("连接阿东 %s", u.String())
	var err error
	c, _, err = websocket.DefaultDialer.Dial(u.String(), http.Header{
		"X-Self-ID":     {fmt.Sprint(jd_cookie.GetInt("selfQid"))},
		"X-Client-Role": {"Universal"},
	})
	if err != nil {
		logs.Warn("连接阿东错误:", err)
		return
	}
	defer c.Close()
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logs.Info("read:", err)
				return
			}

			type AutoGenerated struct {
				Action string `json:"action"`
				Echo   string `json:"echo"`
				Params struct {
					UserID  int64  `json:"user_id"`
					Message string `json:"message"`
				} `json:"params"`
			}
			ag := &AutoGenerated{}
			json.Unmarshal(message, ag)
			// ag.Params.Message = regexp.MustCompile(`\[CQ:[^\[\]]+]`).ReplaceAllString(ag.Params.Message, "")
			if ag.Action == "send_private_msg" {
				if cry, ok := mhome.Load(ag.Params.UserID); ok {
					fmt.Println(ag.Params.Message)
					cry.(chan string) <- ag.Params.Message
				}
			}
			logs.Info("recv: %s", message)
		}
	}()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(`{}`))
			if err != nil {
				logs.Info("阿东错误:", err)
				c = nil
				return
			}
		}
	}
}
