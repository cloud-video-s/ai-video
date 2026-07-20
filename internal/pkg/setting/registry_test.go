package setting

import "testing"

func TestUserSingleDeviceLoginDefinition(t *testing.T) {
	if got := defaultValue(UserSingleDeviceLoginKey); got != "false" {
		t.Fatalf("default value=%q, want false", got)
	}
	for _, item := range registry {
		if item.Key != UserSingleDeviceLoginKey {
			continue
		}
		if item.Group != "用户" || item.Type != "bool" || item.Name != "单设备登录" {
			t.Fatalf("unexpected definition: %+v", item)
		}
		return
	}
	t.Fatalf("configuration %q is not registered", UserSingleDeviceLoginKey)
}

func TestAPPBasicInformationDefinitions(t *testing.T) {
	want := map[string]string{
		APPNameKey: "string", APPAboutKey: "text", APPServicePhoneKey: "string",
		APPServiceEmailKey: "string", APPWebsiteKey: "string", APPThemeColorKey: "color",
		APPThemeModeKey: "select", APPLanguageKey: "select",
	}
	for _, item := range registry {
		wantType, exists := want[item.Key]
		if !exists {
			continue
		}
		if item.Group != "APP 基础信息" || item.Type != wantType || !item.IsPublic {
			t.Fatalf("unexpected APP definition: %+v", item)
		}
		delete(want, item.Key)
	}
	if len(want) != 0 {
		t.Fatalf("missing APP definitions: %v", want)
	}
}
