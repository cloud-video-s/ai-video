package service

import (
	"testing"

	"ai-video/internal/model"
)

func TestShouldCreateDeviceAccount(t *testing.T) {
	email := "user@example.com"
	tests := []struct {
		name     string
		latest   *model.VideoUser
		forceNew bool
		want     bool
	}{
		{name: "first registration", want: true},
		{name: "reuse unbound guest", latest: &model.VideoUser{}, want: false},
		{name: "register after email binding", latest: &model.VideoUser{Email: &email}, want: true},
		{name: "register after account registration", latest: &model.VideoUser{Registered: true}, want: true},
		{name: "force re-register", latest: &model.VideoUser{}, forceNew: true, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldCreateDeviceAccount(test.latest, test.forceNew); got != test.want {
				t.Fatalf("shouldCreateDeviceAccount() = %v, want %v", got, test.want)
			}
		})
	}
}
