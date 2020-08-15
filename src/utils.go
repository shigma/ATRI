package main

import (
	"fmt"
	"hash/crc32"
	"strings"

	"github.com/Mrs4s/MiraiGo/message"
	log "github.com/sirupsen/logrus"
)

func Check(err error) {
	if err != nil {
		log.Fatalf("遇到错误: %v", err)
	}
}

func ToGlobalId(code int64, msgId int32) int32 {
	return int32(crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d-%d", code, msgId))))
}

func ToFormattedMessage(e []message.IMessageElement, code int64, raw ...bool) (r interface{}) {
	r = ToStringMessage(e, code, raw...)
	return
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
