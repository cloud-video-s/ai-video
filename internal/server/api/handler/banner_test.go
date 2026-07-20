package handler

import (
	"net/http/httptest"
	"testing"

	"ai-video/internal/middleware"
	apiservice "ai-video/internal/server/api/server"

	"github.com/gin-gonic/gin"
)

func TestApplyBannerHeadersUsesCurrentAppEnvironment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/api/banners/list?position_key=home", nil)
	req.Header.Set("Video_Device_Country", "CN")
	req.Header.Set("Video_Channel_ID", "channel-a")
	req.Header.Set("Video_Channel_Package", "channel.pkg.a")
	req.Header.Set("Video_App_Package", "app.a")
	req.Header.Set("Video_App_Version", "1.2.3")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Set(middleware.HeaderDeviceCountry, "CN")
	ctx.Set(middleware.HeaderChannelID, "channel-a")
	ctx.Set(middleware.HeaderChannelPackage, "channel.pkg.a")
	ctx.Set(middleware.HeaderAppPackage, "app.a")
	ctx.Set(middleware.HeaderAppVersion, "1.2.3")

	params := &apiservice.ClientBannerRequest{PositionKey: "home"}
	applyBannerHeaders(ctx, params)
	if params.Country != "CN" || params.Channel != "channel-a" || params.ChannelPackage != "channel.pkg.a" ||
		params.PackageCode != "app.a" || params.PackageVersion != "1.2.3" || params.PositionKey != "home" {
		t.Fatalf("header parameters = %#v", params)
	}
}

func TestApplyBannerHeadersUsesAuthenticatedClientHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("GET", "/api/banners", nil)
	req.Header.Set("Video_Device_Country", "US")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Set(middleware.HeaderDeviceCountry, "US")

	params := &apiservice.ClientBannerRequest{Country: "CN"}
	applyBannerHeaders(ctx, params)
	if params.Country != "US" {
		t.Fatalf("country = %q, want authenticated client header", params.Country)
	}
}
