package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-video/internal/app"
)

func TestResolveCountry(t *testing.T) {
	previous := app.Cfg.GeoIP
	t.Cleanup(func() { app.Cfg.GeoIP = previous })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/8.8.8.8" {
			t.Errorf("unexpected lookup path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"country_code":"us"}`))
	}))
	defer server.Close()

	app.Cfg.GeoIP = app.GeoIPConfig{
		LookupURL: server.URL + "/{ip}", CountryField: "country_code", TimeoutMS: 1000,
	}
	country, err := ResolveCountry(context.Background(), "8.8.8.8", "")
	if err != nil {
		t.Fatal(err)
	}
	if country != "US" {
		t.Fatalf("country = %q, want US", country)
	}
}

func TestResolveCountryUsesTrustedHeaderFirst(t *testing.T) {
	country, err := ResolveCountry(context.Background(), "not-an-ip", "cn")
	if err != nil {
		t.Fatal(err)
	}
	if country != "CN" {
		t.Fatalf("country = %q, want CN", country)
	}
}
