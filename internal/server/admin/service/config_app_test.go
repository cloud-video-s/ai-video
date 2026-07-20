package service

import (
	"testing"

	"ai-video/internal/pkg/setting"
)

func TestValidateAPPConfigValues(t *testing.T) {
	valid := map[string]string{
		setting.APPNameKey: "AI Video", setting.APPAboutKey: "About us",
		setting.APPServicePhoneKey: "+86 400-123-4567", setting.APPServiceEmailKey: "support@example.com",
		setting.APPWebsiteKey: "https://example.com/about", setting.APPThemeColorKey: "#409EFF",
		setting.APPThemeModeKey: "system", setting.APPLanguageKey: "zh-CN",
	}
	for key, value := range valid {
		if err := validateConfigValue(key, value); err != nil {
			t.Fatalf("validateConfigValue(%q, %q): %v", key, value, err)
		}
	}

	invalid := map[string]string{
		setting.APPNameKey: "", setting.APPServicePhoneKey: "call-me",
		setting.APPServiceEmailKey: "invalid", setting.APPWebsiteKey: "javascript:alert(1)",
		setting.APPThemeColorKey: "blue", setting.APPThemeModeKey: "neon", setting.APPLanguageKey: "xx-YY",
	}
	for key, value := range invalid {
		if err := validateConfigValue(key, value); err == nil {
			t.Fatalf("validateConfigValue(%q, %q) accepted invalid value", key, value)
		}
	}
}

func TestColorConfigTypeIsSupported(t *testing.T) {
	if err := validateConfigDef("color", ""); err != nil {
		t.Fatal(err)
	}
}
