package main

import (
	// #include "def.h"
	"C"
	"fmt"
	"hash/crc32"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
)
import "strings"

//export _login
func _login(uin C.longlong, pw *C.char) {
	cli := client.NewClient(int64(uin), C.GoString(pw))
	// TODO error handling
	cli.Login()
	NewQQBot(cli)
}

type CQBot struct {
	Client *client.QQClient

	events          []func(MSG)
	friendReqCache  sync.Map
	invitedReqCache sync.Map
	joinReqCache    sync.Map
	tempMsgCache    sync.Map
}

func NewQQBot(cli *client.QQClient) *CQBot {
	bot := &CQBot{
		Client: cli,
	}
	// TODO add event listener somehow
	bot.Client.OnPrivateMessage(bot.privateMessageEvent)
	return bot
}

type MSG map[string]interface{}

func ToGlobalId(code int64, msgId int32) int32 {
	return int32(crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d-%d", code, msgId))))
}

func ToFormattedMessage(e []message.IMessageElement, code int64, raw ...bool) (r interface{}) {
	r = ToStringMessage(e, code, raw...)
	return
}

func (bot *CQBot) privateMessageEvent(c *client.QQClient, m *message.PrivateMessage) {
	// checkMedia(m.Elements)
	cqm := ToStringMessage(m.Elements, 0, true)
	// log.Infof("收到好友 %v(%v) 的消息: %v", m.Sender.DisplayName(), m.Sender.Uin, cqm)
	fm := MSG{
		"post_type":    "message",
		"message_type": "private",
		"sub_type":     "friend",
		"message_id":   ToGlobalId(m.Sender.Uin, m.Id),
		"user_id":      m.Sender.Uin,
		"message":      ToFormattedMessage(m.Elements, 0, false),
		"raw_message":  cqm,
		"font":         0,
		"self_id":      c.Uin,
		"time":         time.Now().Unix(),
		"sender": MSG{
			"user_id":  m.Sender.Uin,
			"nickname": m.Sender.Nickname,
			"sex":      "unknown",
			"age":      0,
		},
	}
	bot.dispatchEventMessage(fm)
}

func (bot *CQBot) dispatchEventMessage(m MSG) {
	for _, f := range bot.events {
		fn := f
		go func() {
			start := time.Now()
			fn(m)
			end := time.Now()
			if end.Sub(start) > time.Second*5 {
				// log.Debugf("警告: 事件处理耗时超过 5 秒 (%v秒), 请检查应用是否有堵塞.", end.Sub(start)/time.Second)
			}
		}()
	}
}

func CQCodeEscapeText(raw string) string {
	ret := raw
	ret = strings.ReplaceAll(ret, "&", "&amp;")
	ret = strings.ReplaceAll(ret, "[", "&#91;")
	ret = strings.ReplaceAll(ret, "]", "&#93;")
	return ret
}

func CQCodeEscapeValue(value string) string {
	ret := CQCodeEscapeText(value)
	ret = strings.ReplaceAll(ret, ",", "&#44;")
	return ret
}

func ToStringMessage(e []message.IMessageElement, code int64, raw ...bool) (r string) {
	ur := false
	if len(raw) != 0 {
		ur = raw[0]
	}
	for _, elem := range e {
		switch o := elem.(type) {
		case *message.TextElement:
			r += CQCodeEscapeText(o.Content)
		case *message.AtElement:
			if o.Target == 0 {
				r += "[CQ:at,qq=all]"
				continue
			}
			r += fmt.Sprintf("[CQ:at,qq=%d]", o.Target)
		case *message.ReplyElement:
			r += fmt.Sprintf("[CQ:reply,id=%d]", ToGlobalId(code, o.ReplySeq))
		case *message.ForwardElement:
			r += fmt.Sprintf("[CQ:forward,id=%s]", o.ResId)
		case *message.FaceElement:
			r += fmt.Sprintf(`[CQ:face,id=%d]`, o.Index)
		case *message.VoiceElement:
			if ur {
				r += fmt.Sprintf(`[CQ:record,file=%s]`, o.Name)
			} else {
				r += fmt.Sprintf(`[CQ:record,file=%s,url=%s]`, o.Name, CQCodeEscapeValue(o.Url))
			}
		case *message.ImageElement:
			if ur {
				r += fmt.Sprintf(`[CQ:image,file=%s]`, o.Filename)
			} else {
				r += fmt.Sprintf(`[CQ:image,file=%s,url=%s]`, o.Filename, CQCodeEscapeValue(o.Url))
			}
		}
	}
	return
}

func main() {
}
