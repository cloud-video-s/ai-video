package adjustevent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/pkg/adjust"
)

const (
	messageKindTrigger = "trigger"
	messageKindReport  = "report"
)

// Message is the durable Redis task payload used by the Adjust event pipeline.
// UserID, Action and ChannelID are the required business fields. The remaining
// fields snapshot rule inputs so delayed processing observes the operation as
// it happened rather than newer user/order state.
type Message struct {
	Kind         string            `json:"kind"`
	EventID      string            `json:"event_id"`
	ParentID     string            `json:"parent_id,omitempty"`
	UserID       uint64            `json:"user_id"`
	Action       adjust.EventToken `json:"action"`
	ChannelID    uint64            `json:"channel_id"`
	ReportAction adjust.EventToken `json:"report_action,omitempty"`
	OrderNo      string            `json:"order_no,omitempty"`
	OrderCount   uint64            `json:"order_count,omitempty"`
	OccurredAt   time.Time         `json:"occurred_at"`
	NotBefore    time.Time         `json:"not_before,omitempty"`
	Revenue      *float64          `json:"revenue,omitempty"`
	Currency     string            `json:"currency,omitempty"`
}

type EnqueueOptions struct {
	OrderNo    string
	OccurredAt time.Time
}

func (message *Message) normalize() {
	message.Kind = strings.ToLower(strings.TrimSpace(message.Kind))
	message.EventID = strings.TrimSpace(message.EventID)
	message.ParentID = strings.TrimSpace(message.ParentID)
	message.OrderNo = strings.TrimSpace(message.OrderNo)
	message.Currency = strings.ToUpper(strings.TrimSpace(message.Currency))
	if message.OccurredAt.IsZero() {
		message.OccurredAt = time.Now()
	}
}

func (message Message) validate() error {
	if message.Kind != messageKindTrigger && message.Kind != messageKindReport {
		return fmt.Errorf("unsupported Adjust event message kind %q", message.Kind)
	}
	if message.EventID == "" || len(message.EventID) > 64 {
		return errors.New("Adjust event message ID is required and must not exceed 64 bytes")
	}
	if message.UserID == 0 {
		return errors.New("Adjust event user ID is required")
	}
	if _, ok := triggerName(message.Action); !ok {
		return fmt.Errorf("unsupported Adjust trigger action %q", message.Action)
	}
	if message.Kind == messageKindReport {
		if message.ParentID == "" {
			return errors.New("Adjust report message parent ID is required")
		}
		if _, ok := callbackName(message.ReportAction); !ok {
			return fmt.Errorf("unsupported Adjust report action %q", message.ReportAction)
		}
	}
	return nil
}

func triggerEventID(userID uint64, action adjust.EventToken, orderNo string) string {
	return stableEventID("trigger", fmt.Sprint(userID), string(action), strings.TrimSpace(orderNo))
}

func reportEventID(parentID string, action adjust.EventToken) string {
	return stableEventID("report", parentID, string(action))
}

func stableEventID(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func triggerName(token adjust.EventToken) (string, bool) {
	switch token {
	case adjust.EventTokenPayment:
		return callbackPayment, true
	case adjust.EventTokenOrderCreated:
		return callbackOrderCreated, true
	case adjust.EventTokenActivation:
		return callbackActivation, true
	case adjust.EventTokenLogin:
		return callbackLogin, true
	case adjust.EventTokenSubscription:
		return callbackSubscription, true
	default:
		return "", false
	}
}

func callbackName(token adjust.EventToken) (string, bool) {
	return triggerName(token)
}

func callbackToken(name string) (adjust.EventToken, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case callbackPayment:
		return adjust.EventTokenPayment, true
	case callbackOrderCreated:
		return adjust.EventTokenOrderCreated, true
	case callbackActivation:
		return adjust.EventTokenActivation, true
	case callbackLogin:
		return adjust.EventTokenLogin, true
	case callbackSubscription:
		return adjust.EventTokenSubscription, true
	default:
		return "", false
	}
}
