package service

import (
	"testing"

	"ai-video/internal/model"
)

func validChannelPayload() ChannelPayload {
	return ChannelPayload{
		ChannelCode: "META_US_001", ChannelName: "Meta 美国渠道", AgencyCompany: "示例代理",
		AdPlatform: "Meta Ads", DeliveryPackage: "com.example.app",
		TrackingURL: "https://tracker.example.com/click?channel=meta", PortRebate: 12.5,
		ServiceOrderFee: 0.35, UploadMethod: "API", Status: 1,
	}
}

func TestValidateChannelPayloadFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ChannelPayload)
		wantErr bool
	}{
		{name: "valid"},
		{name: "invalid code", mutate: func(p *ChannelPayload) { p.ChannelCode = "META US" }, wantErr: true},
		{name: "empty name", mutate: func(p *ChannelPayload) { p.ChannelName = " " }, wantErr: true},
		{name: "negative rebate", mutate: func(p *ChannelPayload) { p.PortRebate = -0.1 }, wantErr: true},
		{name: "rebate over one hundred", mutate: func(p *ChannelPayload) { p.PortRebate = 100.01 }, wantErr: true},
		{name: "negative service fee", mutate: func(p *ChannelPayload) { p.ServiceOrderFee = -0.01 }, wantErr: true},
		{name: "service fee too large", mutate: func(p *ChannelPayload) { p.ServiceOrderFee = 10000000000 }, wantErr: true},
		{name: "invalid tracking URL", mutate: func(p *ChannelPayload) { p.TrackingURL = "javascript:alert(1)" }, wantErr: true},
		{name: "empty tracking URL", mutate: func(p *ChannelPayload) { p.TrackingURL = "" }},
		{name: "extensible upload method", mutate: func(p *ChannelPayload) { p.UploadMethod = "CUSTOM_API" }},
		{name: "invalid upload method", mutate: func(p *ChannelPayload) { p.UploadMethod = "API PUSH" }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validChannelPayload()
			if tt.mutate != nil {
				tt.mutate(&payload)
			}
			err := validateChannelPayloadFields(&payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateChannelPayloadFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyChannelPayloadNormalizesValues(t *testing.T) {
	payload := validChannelPayload()
	payload.ChannelCode = " META_US_001 "
	payload.ChannelName = " Meta 美国渠道 "
	payload.UploadMethod = " api "
	item := &model.VideoChannel{}
	applyChannelPayload(item, &payload)
	if item.ChannelCode != "META_US_001" {
		t.Fatalf("ChannelCode = %q", item.ChannelCode)
	}
	if item.ChannelName != "Meta 美国渠道" {
		t.Fatalf("ChannelName = %q", item.ChannelName)
	}
	if item.UploadMethod != "API" {
		t.Fatalf("UploadMethod = %q", item.UploadMethod)
	}
}
