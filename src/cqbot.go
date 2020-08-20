package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	log "github.com/sirupsen/logrus"
	asciiart "github.com/yinghau76/go-ascii-art"
)

type CQBot struct {
	Client *client.QQClient

	events          []func(MSG)
	friendReqCache  sync.Map
	invitedReqCache sync.Map
	joinReqCache    sync.Map
	tempMsgCache    sync.Map
}
type MSG map[string]interface{}

func SetupDevice(info string) {
	// "{\"display\":\"MIRAI.991110.001\",\"finger_print\":\"mamoe/mirai/mirai:10/MIRAI.200122.001/3854695:user/release-keys\",\"boot_id\":\"3B51B494-F2B9-6577-045F-D9CC60EAAB2C\",\"proc_version\":\"Linux version 3.0.31-BOECBqqM (android-build@xxx.xxx.xxx.xxx.com)\",\"imei\":\"116708152627273\"}"
	client.SystemDeviceInfo.ReadJson([]byte(info))
}

func NewCQBot(uin int64, pw string) *CQBot {
	cli := client.NewClient(uin, pw)
	return &CQBot{
		Client: cli,
	}
}

func (bot *CQBot) Login() bool {
	cli := bot.Client
	rsp, err := cli.Login()
	Check(err)
	if !rsp.Success {
		panic(rsp)
	}
	log.Info("开始加载好友列表...")
	Check(cli.ReloadFriendList())
	log.Infof("共加载 %v 个好友.", len(cli.FriendList))
	log.Infof("开始加载群列表...")
	Check(cli.ReloadGroupList(true))
	log.Infof("共加载 %v 个群.", len(cli.GroupList))
	log.Infof("登录成功: %v", cli.Nickname)
	return true
}

func (bot *CQBot) LoginInteractive() bool {
	console := bufio.NewReader(os.Stdin)
	cli := bot.Client
	// TODO error handling
	rsp, err := cli.Login()
	for {
		Check(err)
		if !rsp.Success {
			switch rsp.Error {
			case client.NeedCaptcha:
				_ = ioutil.WriteFile("captcha.jpg", rsp.CaptchaImage, os.ModePerm)
				img, _, _ := image.Decode(bytes.NewReader(rsp.CaptchaImage))
				fmt.Println(asciiart.New("image", img).Art)
				log.Warn("请输入验证码 (captcha.jpg)： (Enter 提交)")
				text, _ := console.ReadString('\n')
				rsp, err = cli.SubmitCaptcha(strings.ReplaceAll(text, "\n", ""), rsp.CaptchaSign)
				continue
			case client.UnsafeDeviceError:
				log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证并重启Bot.", rsp.VerifyUrl)
				log.Infof(" 按 Enter 继续....")
				_, _ = console.ReadString('\n')
				return false
			case client.OtherLoginError, client.UnknownLoginError:
				log.Fatalf("登录失败: %v", rsp.ErrorMessage)
			}
		}
		break
	}
	log.Info("开始加载好友列表...")
	Check(cli.ReloadFriendList())
	log.Infof("共加载 %v 个好友.", len(cli.FriendList))
	log.Infof("开始加载群列表...")
	Check(cli.ReloadGroupList())
	log.Infof("共加载 %v 个群.", len(cli.GroupList))
	log.Infof("登录成功: %v", cli.Nickname)
	return true
}

func (bot *CQBot) onEvent(callback func(MSG)) {
	bot.Client.OnPrivateMessage(func(c *client.QQClient, m *message.PrivateMessage) {
		cqm := ToStringMessage(m.Elements, 0, true)
		callback(MSG{
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
		})
	})

	bot.Client.OnGroupMessage(func(c *client.QQClient, m *message.GroupMessage) {
		for _, elem := range m.Elements {
			if file, ok := elem.(*message.GroupFileElement); ok {
				callback(MSG{
					"post_type":   "notice",
					"notice_type": "group_upload",
					"group_id":    m.GroupCode,
					"user_id":     m.Sender.Uin,
					"file": MSG{
						"id":    file.Path,
						"name":  file.Name,
						"size":  file.Size,
						"busid": file.Busid,
						"url":   c.GetGroupFileUrl(m.GroupCode, file.Path, file.Busid),
					},
					"self_id": c.Uin,
					"time":    time.Now().Unix(),
				})
				return
			}
		}
		cqm := ToStringMessage(m.Elements, m.GroupCode, true)
		id := m.Id
		// TODO db
		// if bot.db != nil {
		// 	id = bot.InsertGroupMessage(m)
		// }
		gm := MSG{
			"anonymous":    nil,
			"font":         0,
			"group_id":     m.GroupCode,
			"message":      ToFormattedMessage(m.Elements, m.GroupCode, false),
			"message_id":   id,
			"message_type": "group",
			"post_type":    "message",
			"raw_message":  cqm,
			"self_id":      c.Uin,
			"sender": MSG{
				"age":     0,
				"area":    "",
				"level":   "",
				"sex":     "unknown",
				"user_id": m.Sender.Uin,
			},
			"sub_type": "normal",
			"time":     time.Now().Unix(),
			"user_id":  m.Sender.Uin,
		}
		if m.Sender.IsAnonymous() {
			gm["anonymous"] = MSG{
				"flag": "",
				"id":   0,
				"name": m.Sender.Nickname,
			}
			gm["sender"].(MSG)["nickname"] = "匿名消息"
			gm["sub_type"] = "anonymous"
		} else {
			mem := c.FindGroup(m.GroupCode).FindMember(m.Sender.Uin)
			ms := gm["sender"].(MSG)
			ms["role"] = func() string {
				switch mem.Permission {
				case client.Owner:
					return "owner"
				case client.Administrator:
					return "admin"
				default:
					return "member"
				}
			}()
			ms["nickname"] = mem.Nickname
			ms["card"] = mem.CardName
			ms["title"] = mem.SpecialTitle
		}
		callback(gm)
	})

	bot.Client.OnTempMessage(func(c *client.QQClient, m *message.TempMessage) {
		cqm := ToStringMessage(m.Elements, 0, true)
		bot.tempMsgCache.Store(m.Sender.Uin, m.GroupCode)
		callback(MSG{
			"post_type":    "message",
			"message_type": "private",
			"sub_type":     "group",
			"message_id":   m.Id,
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
		})
	})

	bot.Client.OnGroupMuted(func(c *client.QQClient, e *client.GroupMuteEvent) {
		callback(MSG{
			"post_type":   "notice",
			"duration":    e.Time,
			"group_id":    e.GroupCode,
			"notice_type": "group_ban",
			"operator_id": e.OperatorUin,
			"self_id":     c.Uin,
			"user_id":     e.TargetUin,
			"time":        time.Now().Unix(),
			"sub_type": func() string {
				if e.Time > 0 {
					return "ban"
				}
				return "lift_ban"
			}(),
		})
	})

	bot.Client.OnFriendMessageRecalled(func(c *client.QQClient, e *client.FriendMessageRecalledEvent) {
		f := c.FindFriend(e.FriendUin)
		gid := ToGlobalId(e.FriendUin, e.MessageId)
		callback(MSG{
			"post_type":   "notice",
			"notice_type": "friend_recall",
			"self_id":     c.Uin,
			"user_id":     f.Uin,
			"time":        e.Time,
			"message_id":  gid,
		})
	})

	bot.Client.OnGroupMessageRecalled(func(c *client.QQClient, e *client.GroupMessageRecalledEvent) {
		gid := ToGlobalId(e.GroupCode, e.MessageId)
		callback(MSG{
			"post_type":   "notice",
			"group_id":    e.GroupCode,
			"notice_type": "group_recall",
			"self_id":     c.Uin,
			"user_id":     e.AuthorUin,
			"operator_id": e.OperatorUin,
			"time":        e.Time,
			"message_id":  gid,
		})
	})

	bot.Client.OnJoinGroup(func(c *client.QQClient, group *client.GroupInfo) {
		callback(MSG{
			"post_type":   "notice",
			"notice_type": "group_increase",
			"group_id":    group.Code,
			"operator_id": 0,
			"self_id":     bot.Client.Uin,
			"sub_type":    "approve",
			"time":        time.Now().Unix(),
			"user_id":     c.Uin,
		})
	})

	bot.Client.OnGroupMemberJoined(func(c *client.QQClient, e *client.MemberJoinGroupEvent) {
		callback(MSG{
			"post_type":   "notice",
			"notice_type": "group_increase",
			"group_id":    e.Group.Code,
			"operator_id": 0,
			"self_id":     bot.Client.Uin,
			"sub_type":    "approve",
			"time":        time.Now().Unix(),
			"user_id":     e.Member.Uin,
		})
	})

	bot.Client.OnLeaveGroup(func(c *client.QQClient, e *client.GroupLeaveEvent) {
		callback(MSG{
			"post_type":   "notice",
			"notice_type": "group_decrease",
			"group_id":    e.Group.Code,
			"operator_id": func() int64 {
				if e.Operator != nil {
					return e.Operator.Uin
				}
				return c.Uin
			}(),
			"self_id": bot.Client.Uin,
			"sub_type": func() string {
				if e.Operator != nil {
					if c.Uin == bot.Client.Uin {
						return "kick_me"
					}
					return "kick"
				}
				return "leave"
			}(),
			"time":    time.Now().Unix(),
			"user_id": c.Uin,
		})
	})

	bot.Client.OnGroupMemberLeaved(func(c *client.QQClient, e *client.MemberLeaveGroupEvent) {
		callback(MSG{
			"post_type":   "notice",
			"notice_type": "group_decrease",
			"group_id":    e.Group.Code,
			"operator_id": func() int64 {
				if e.Operator != nil {
					return e.Operator.Uin
				}
				return e.Member.Uin
			}(),
			"self_id": bot.Client.Uin,
			"sub_type": func() string {
				if e.Operator != nil {
					if e.Member.Uin == bot.Client.Uin {
						return "kick_me"
					}
					return "kick"
				}
				return "leave"
			}(),
			"time":    time.Now().Unix(),
			"user_id": e.Member.Uin,
		})
	})

	bot.Client.OnGroupMuted(func(c *client.QQClient, e *client.GroupMuteEvent) {
		callback(MSG{
			"post_type":   "notice",
			"duration":    e.Time,
			"group_id":    e.GroupCode,
			"notice_type": "group_ban",
			"operator_id": e.OperatorUin,
			"self_id":     c.Uin,
			"user_id":     e.TargetUin,
			"time":        time.Now().Unix(),
			"sub_type": func() string {
				if e.Time > 0 {
					return "ban"
				}
				return "lift_ban"
			}(),
		})
	})

	bot.Client.OnGroupMemberPermissionChanged(func(c *client.QQClient, e *client.MemberPermissionChangedEvent) {
		callback(MSG{
			"post_type":   "notice",
			"notice_type": "group_admin",
			"sub_type": func() string {
				if e.NewPermission == client.Administrator {
					return "set"
				}
				return "unset"
			}(),
			"group_id": e.Group.Code,
			"user_id":  e.Member.Uin,
			"time":     time.Now().Unix(),
			"self_id":  c.Uin,
		})
	})

	bot.Client.OnNewFriendRequest(func(c *client.QQClient, e *client.NewFriendRequest) {
		flag := strconv.FormatInt(e.RequestId, 10)
		bot.friendReqCache.Store(flag, e)
		callback(MSG{
			"post_type":    "request",
			"request_type": "friend",
			"user_id":      e.RequesterUin,
			"comment":      e.Message,
			"flag":         flag,
			"time":         time.Now().Unix(),
			"self_id":      c.Uin,
		})
	})

	bot.Client.OnNewFriendAdded(func(c *client.QQClient, e *client.NewFriendEvent) {
		bot.tempMsgCache.Delete(e.Friend.Uin)
		callback(MSG{
			"post_type":   "notice",
			"notice_type": "friend_add",
			"self_id":     c.Uin,
			"user_id":     e.Friend.Uin,
			"time":        time.Now().Unix(),
		})
	})

	bot.Client.OnGroupInvited(func(c *client.QQClient, e *client.GroupInvitedRequest) {
		flag := strconv.FormatInt(e.RequestId, 10)
		bot.invitedReqCache.Store(flag, e)
		callback(MSG{
			"post_type":    "request",
			"request_type": "group",
			"sub_type":     "invite",
			"group_id":     e.GroupCode,
			"user_id":      e.InvitorUin,
			"comment":      "",
			"flag":         flag,
			"time":         time.Now().Unix(),
			"self_id":      c.Uin,
		})
	})

	bot.Client.OnUserWantJoinGroup(func(c *client.QQClient, e *client.UserJoinGroupRequest) {
		flag := strconv.FormatInt(e.RequestId, 10)
		bot.joinReqCache.Store(flag, e)
		callback(MSG{
			"post_type":    "request",
			"request_type": "group",
			"sub_type":     "add",
			"group_id":     e.GroupCode,
			"user_id":      e.RequesterUin,
			"comment":      e.Message,
			"flag":         flag,
			"time":         time.Now().Unix(),
			"self_id":      c.Uin,
		})
	})
}

func (bot *CQBot) _SendPrivateMessage(target int64, m *message.SendingMessage) int32 {
	var newElem []message.IMessageElement
	for _, elem := range m.Elements {
		if i, ok := elem.(*message.ImageElement); ok {
			fm, err := bot.Client.UploadPrivateImage(target, i.Data)
			if err != nil {
				log.Warnf("警告: 私聊 %v 消息图片上传失败.", target)
				continue
			}
			newElem = append(newElem, fm)
			continue
		}
		newElem = append(newElem, elem)
	}
	m.Elements = newElem
	var id int32
	if bot.Client.FindFriend(target) != nil {
		id = bot.Client.SendPrivateMessage(target, m).Id
	} else {
		if code, ok := bot.tempMsgCache.Load(target); ok {
			id = bot.Client.SendTempMessage(code.(int64), target, m).Id
		} else {
			return -1
		}
	}
	return ToGlobalId(target, id)
}

func (bot *CQBot) SendPrivateMessage(userId int64, content string) MSG {
	elem := bot.ConvertStringMessage(content, false)
	mid := bot._SendPrivateMessage(userId, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return nil
	}
	return MSG{"message_id": mid}
}

func (bot *CQBot) _SendGroupMessage(groupId int64, m *message.SendingMessage) int32 {
	var newElem []message.IMessageElement
	for _, elem := range m.Elements {
		if i, ok := elem.(*message.ImageElement); ok {
			_, _ = bot.Client.UploadGroupImage(int64(rand.Intn(11451419)), i.Data)
			gm, err := bot.Client.UploadGroupImage(groupId, i.Data)
			if err != nil {
				log.Warnf("警告: 群 %v 消息图片上传失败: %v", groupId, err)
				continue
			}
			newElem = append(newElem, gm)
			continue
		}
		if i, ok := elem.(*message.VoiceElement); ok {
			gv, err := bot.Client.UploadGroupPtt(groupId, i.Data)
			if err != nil {
				log.Warnf("警告: 群 %v 消息语音上传失败: %v", groupId, err)
				continue
			}
			newElem = append(newElem, gv)
			continue
		}
		newElem = append(newElem, elem)
	}
	m.Elements = newElem
	ret := bot.Client.SendGroupMessage(groupId, m)
	if ret.Id == -1 {
		return -1
	}
	// return bot.InsertGroupMessage(ret)
	return 0
}

func (bot *CQBot) SendGroupMessage(groupId int64, i interface{}) MSG {
	var str string
	if s, ok := i.(string); ok {
		str = s
	}
	if str == "" {
		return nil
	}
	elem := bot.ConvertStringMessage(str, true)
	fixAt := func(elem []message.IMessageElement) {
		for _, e := range elem {
			if at, ok := e.(*message.AtElement); ok && at.Target != 0 {
				at.Display = "@" + func() string {
					mem := bot.Client.FindGroup(groupId).FindMember(at.Target)
					if mem != nil {
						return mem.DisplayName()
					}
					return strconv.FormatInt(at.Target, 10)
				}()
			}
		}
	}
	fixAt(elem)
	mid := bot._SendGroupMessage(groupId, &message.SendingMessage{Elements: elem})
	if mid == -1 {
		return nil
	}
	return MSG{"message_id": mid}
}

// GetGroupMemberList
func (bot *CQBot) GetGroupMemberList(groupId int64) []MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		return nil
	}
	var members []MSG
	for _, m := range group.Members {
		members = append(members, convertGroupMemberInfo(groupId, m))
	}
	return members
}

func convertGroupMemberInfo(groupId int64, m *client.GroupMemberInfo) MSG {
	return MSG{
		"group_id":       groupId,
		"user_id":        m.Uin,
		"nickname":       m.Nickname,
		"card":           m.CardName,
		"sex":            "unknown",
		"age":            0,
		"area":           "",
		"join_time":      m.JoinTime,
		"last_sent_time": m.LastSpeakTime,
		"level":          strconv.FormatInt(int64(m.Level), 10),
		"role": func() string {
			switch m.Permission {
			case client.Owner:
				return "owner"
			case client.Administrator:
				return "admin"
			default:
				return "member"
			}
		}(),
		"unfriendly":        false,
		"title":             m.SpecialTitle,
		"title_expire_time": m.SpecialTitleExpireTime,
		"card_changeable":   false,
	}
}

// GetGroupInfo
func (bot *CQBot) GetGroupInfo(groupId int64) MSG {
	group := bot.Client.FindGroup(groupId)
	if group == nil {
		return nil
	}
	return MSG{
		"group_id":         group.Code,
		"group_name":       group.Name,
		"max_member_count": group.MaxMemberCount,
		"member_count":     group.MemberCount,
	}
}

// GetGroupList
func (bot *CQBot) GetGroupList() []MSG {
	var gs []MSG
	for _, g := range bot.Client.GroupList {
		gs = append(gs, MSG{
			"group_id":         g.Code,
			"group_name":       g.Name,
			"max_member_count": g.MaxMemberCount,
			"member_count":     g.MemberCount,
		})
	}
	return gs
}

//export getFriendList
func (bot *CQBot) GetFriendList() []MSG {
	var fs []MSG
	for _, f := range bot.Client.FriendList {
		fs = append(fs, MSG{
			"nickname": f.Nickname,
			"remark":   f.Remark,
			"user_id":  f.Uin,
		})
	}
	return fs
}
