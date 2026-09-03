package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var url = "https://oapi.dingtalk.com/robot/send?access_token=69620325d001b394f2ba2f0accb626ea4eaef7efc7dbad770f3a001c24a419d9"
var secret = "SEC82c4efa942d107e09abb7f04978d4cca3db512a0cb6c44fbe8a5e3b2d0681588"

// 消息类型常量
const (
	MsgTypeText       = "text"
	MsgTypeMarkdown   = "markdown"
	MsgTypeLink       = "link"
	MsgTypeActionCard = "actionCard"
	MsgTypeFeedCard   = "feedCard"
)

// Robot 钉钉机器人客户端
type Robot struct {
	Webhook string // 机器人 Webhook 地址
	Secret  string // 加签密钥（可选，开启加签时必填）
	Client  *http.Client
}

// NewRobot 创建机器人客户端
func NewRobot(webhook, secret string) *Robot {
	return &Robot{
		Webhook: webhook,
		Secret:  secret,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GenerateSignedURL 生成带签名的 Webhook URL
// 钉钉机器人安全设置中的"加签"方式需要使用此方法[reference:0][reference:1]
func (r *Robot) GenerateSignedURL() (string, error) {
	if r.Webhook == "" {
		return "", fmt.Errorf("webhook 不能为空")
	}
	if r.Secret == "" {
		return r.Webhook, nil // 未开启加签，直接返回原 URL
	}

	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	// 待签名字符串：timestamp + "\n" + secret[reference:2]
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, r.Secret)

	h := hmac.New(sha256.New, []byte(r.Secret))
	h.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%s×tamp=%s&sign=%s", r.Webhook, timestamp, sign), nil
}

// Send 发送消息
func (r *Robot) Send(msg interface{}) (map[string]interface{}, error) {
	url, err := r.GenerateSignedURL()
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("序列化消息失败: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result map[string]interface{}
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查钉钉返回的 errcode[reference:3]
	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		errmsg, _ := result["errmsg"].(string)
		return nil, fmt.Errorf("钉钉返回错误: errcode=%v, errmsg=%s", errcode, errmsg)
	}

	return result, nil
}

// ============ 各类消息构造 ============

// TextMessage 文本消息[reference:4]
type TextMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	At *AtInfo `json:"at,omitempty"`
}

// AtInfo @ 人信息
type AtInfo struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtUserIds []string `json:"atUserIds,omitempty"`
	IsAtAll   bool     `json:"isAtAll"`
}

// NewTextMessage 创建文本消息
func NewTextMessage(content string, atMobiles []string, isAtAll bool) *TextMessage {
	msg := &TextMessage{MsgType: MsgTypeText}
	msg.Text.Content = content
	if len(atMobiles) > 0 || isAtAll {
		msg.At = &AtInfo{
			AtMobiles: atMobiles,
			IsAtAll:   isAtAll,
		}
	}
	return msg
}

// MarkdownMessage Markdown 消息[reference:6][reference:7]
type MarkdownMessage struct {
	MsgType  string `json:"msgtype"`
	Markdown struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"markdown"`
	At *AtInfo `json:"at,omitempty"`
}

// NewMarkdownMessage 创建 Markdown 消息
func NewMarkdownMessage(title, text string, atMobiles []string, isAtAll bool) *MarkdownMessage {
	msg := &MarkdownMessage{MsgType: MsgTypeMarkdown}
	msg.Markdown.Title = title
	msg.Markdown.Text = text
	if len(atMobiles) > 0 || isAtAll {
		msg.At = &AtInfo{
			AtMobiles: atMobiles,
			IsAtAll:   isAtAll,
		}
	}
	return msg
}

// LinkMessage 链接消息[reference:8]
type LinkMessage struct {
	MsgType string `json:"msgtype"`
	Link    struct {
		Text       string `json:"text"`
		Title      string `json:"title"`
		PicUrl     string `json:"picUrl,omitempty"`
		MessageUrl string `json:"messageUrl"`
	} `json:"link"`
}

// NewLinkMessage 创建链接消息
func NewLinkMessage(title, text, messageUrl, picUrl string) *LinkMessage {
	msg := &LinkMessage{MsgType: MsgTypeLink}
	msg.Link.Title = title
	msg.Link.Text = text
	msg.Link.MessageUrl = messageUrl
	msg.Link.PicUrl = picUrl
	return msg
}

// SendText 快捷发送文本消息
func (r *Robot) SendText(content string, atMobiles []string, isAtAll bool) (map[string]interface{}, error) {
	msg := NewTextMessage(content, atMobiles, isAtAll)
	return r.Send(msg)
}

// SendMarkdown 快捷发送 Markdown 消息
func (r *Robot) SendMarkdown(title, content string, atMobiles []string, isAtAll bool) (map[string]interface{}, error) {
	msg := NewMarkdownMessage(title, content, atMobiles, isAtAll)
	return r.Send(msg)
}

// SendLink 快捷发送链接消息
func (r *Robot) SendLink(title, text, messageUrl, picUrl string) (map[string]interface{}, error) {
	msg := NewLinkMessage(title, text, messageUrl, picUrl)
	return r.Send(msg)
}
