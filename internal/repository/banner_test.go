package repository

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"ai-video/internal/app"
	"ai-video/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBannerListForClientAppliesDeliveryTargets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:client-banner-targets?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.AutoMigrate(
		&model.VideoTemplateType{}, &model.VideoCountry{}, &model.VideoPackage{},
		&model.VideoChannel{}, &model.VideoDisplayPosition{}, &model.VideoTemplate{}, &model.VideoBanner{},
	); err != nil {
		t.Fatal(err)
	}

	cn := model.VideoCountry{Code: "CN", NameZh: "China", Status: 1}
	us := model.VideoCountry{Code: "US", NameZh: "United States", Status: 1}
	channelA := model.VideoChannel{ChannelCode: "channel-a", ChannelName: "Channel A", AdPlatform: "direct", DeliveryPackage: "channel.pkg.a", UploadMethod: "API", Status: 1}
	channelB := model.VideoChannel{ChannelCode: "channel-b", ChannelName: "Channel B", AdPlatform: "direct", DeliveryPackage: "channel.pkg.b", UploadMethod: "API", Status: 1}
	packageA := model.VideoPackage{PackageName: "App A", PackageCode: "app.a", PackageVersion: "1.0.0", DownloadURL: "https://example.com/a", Status: 1}
	packageB := model.VideoPackage{PackageName: "App B", PackageCode: "app.b", PackageVersion: "1.0.0", DownloadURL: "https://example.com/b", Status: 1}
	templateType := model.VideoTemplateType{CategoryName: "General", Status: 1}
	homePosition := model.VideoDisplayPosition{PositionName: "Home", PositionKey: "home", CoverImage: "https://example.com/home.jpg", Status: 1}
	secondaryPosition := model.VideoDisplayPosition{PositionName: "Secondary", PositionKey: "secondary", CoverImage: "https://example.com/secondary.jpg", Status: 1}
	disabledPosition := model.VideoDisplayPosition{PositionName: "Disabled", PositionKey: "disabled", CoverImage: "https://example.com/disabled-position.jpg", Status: 1}
	for _, item := range []interface{}{&cn, &us, &channelA, &channelB, &packageA, &packageB, &templateType, &homePosition, &secondaryPosition, &disabledPosition} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&disabledPosition).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}

	activeTemplate := model.VideoTemplate{
		VideoTemplateTypeID: templateType.ID, Name: "Active template", TemplateType: model.VideoTemplateKindAction,
		CoverImage: "https://example.com/template.jpg", TemplateVideo: "https://example.com/template.mp4", Status: 1,
	}
	disabledTemplate := model.VideoTemplate{
		VideoTemplateTypeID: templateType.ID, Name: "Disabled template", TemplateType: model.VideoTemplateKindAction,
		CoverImage: "https://example.com/disabled.jpg", TemplateVideo: "https://example.com/disabled.mp4", Status: 0,
	}
	if err := db.Create(&activeTemplate).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&disabledTemplate).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&disabledTemplate).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewBannerRepo()
	createBanner := func(name, positionKey string, status int8, sort uint64, jumpType uint8, templateID *uint64, targets BannerTargetIDs) {
		t.Helper()
		item := model.VideoBanner{
			Name: name, CoverImage: "https://example.com/" + name + ".jpg", Sort: sort,
			JumpType: jumpType, TemplateID: templateID, Status: status,
		}
		if len(targets.DisplayPositionKeys) == 0 {
			switch positionKey {
			case "home":
				targets.DisplayPositionKeys = []string{homePosition.PositionKey}
			case "secondary":
				targets.DisplayPositionKeys = []string{secondaryPosition.PositionKey}
			case "disabled":
				targets.DisplayPositionKeys = []string{disabledPosition.PositionKey}
			}
		}
		if err := repo.Create(context.Background(), &item); err != nil {
			t.Fatal(err)
		}
		if err := repo.ReplaceTargets(context.Background(), &item, targets); err != nil {
			t.Fatal(err)
		}
	}

	createBanner("global", "home", 1, 0, model.BannerJumpTypeLink, nil, BannerTargetIDs{
		DisplayPositionKeys: []string{homePosition.PositionKey, secondaryPosition.PositionKey},
	})
	createBanner("matched", "home", 1, 1, model.BannerJumpTypeLink, nil, BannerTargetIDs{
		CountryIDs: []uint64{cn.ID}, ChannelIDs: []uint64{channelA.ChannelID}, PackageIDs: []uint64{packageA.ID},
	})
	createBanner("wrong-country", "home", 1, 2, model.BannerJumpTypeLink, nil, BannerTargetIDs{CountryIDs: []uint64{us.ID}})
	createBanner("wrong-channel", "home", 1, 3, model.BannerJumpTypeLink, nil, BannerTargetIDs{ChannelIDs: []uint64{channelB.ChannelID}})
	createBanner("wrong-package", "home", 1, 4, model.BannerJumpTypeLink, nil, BannerTargetIDs{PackageIDs: []uint64{packageB.ID}})
	createBanner("disabled", "home", 0, 5, model.BannerJumpTypeLink, nil, BannerTargetIDs{})
	createBanner("active-template", "home", 1, 6, model.BannerJumpTypeTemplate, &activeTemplate.ID, BannerTargetIDs{
		CountryIDs: []uint64{cn.ID}, ChannelIDs: []uint64{channelA.ChannelID}, PackageIDs: []uint64{packageA.ID},
	})
	createBanner("disabled-template", "home", 1, 7, model.BannerJumpTypeTemplate, &disabledTemplate.ID, BannerTargetIDs{})
	createBanner("other-position", "secondary", 1, 8, model.BannerJumpTypeLink, nil, BannerTargetIDs{})
	createBanner("disabled-position", "disabled", 1, 9, model.BannerJumpTypeLink, nil, BannerTargetIDs{})
	if err := db.Model(&model.VideoBanner{}).Where("name = ?", "disabled").Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListForClient(context.Background(), ClientBannerTargets{
		PositionKey: "home", CountryID: cn.ID, ChannelIDs: []uint64{channelA.ChannelID}, PackageIDs: []uint64{packageA.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := bannerNames(list), []string{"global", "matched", "active-template"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("banner names = %v, want %v", got, want)
	}
	if list[2].Template == nil || list[2].Template.ID != activeTemplate.ID {
		t.Fatalf("active template was not preloaded: %#v", list[2].Template)
	}

	globalOnly, err := repo.ListForClient(context.Background(), ClientBannerTargets{PositionKey: "home"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := bannerNames(globalOnly), []string{"global"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("global banner names = %v, want %v", got, want)
	}

	secondary, err := repo.ListForClient(context.Background(), ClientBannerTargets{PositionKey: "secondary"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := bannerNames(secondary), []string{"global", "other-position"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("secondary banner names = %v, want %v", got, want)
	}
}

func bannerNames(items []model.VideoBanner) []string {
	names := make([]string, len(items))
	for i := range items {
		names[i] = items[i].Name
	}
	return names
}

func TestChannelResolveEnabledTargetsUsesAllProvidedValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:client-banner-channel?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.AutoMigrate(&model.VideoChannel{}); err != nil {
		t.Fatal(err)
	}
	items := []model.VideoChannel{
		{ChannelCode: "same-code", ChannelName: "A", AdPlatform: "direct", DeliveryPackage: "pkg.a", UploadMethod: "API", Status: 1},
		{ChannelCode: "other-code", ChannelName: "B", AdPlatform: "direct", DeliveryPackage: "pkg.b", UploadMethod: "API", Status: 1},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatal(err)
	}

	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	list, err := NewChannelRepo().ResolveEnabledTargets(ginContext, "same-code", "pkg.b")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("mismatched channel code and delivery package returned %d rows", len(list))
	}
}
