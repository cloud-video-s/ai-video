package repository

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"ai-video/internal/app"
	"ai-video/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTemplateTypeListForClientUsesPositionKeyAndDeliveryTargets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:client-template-types?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.AutoMigrate(&model.VideoDisplayPosition{}, &model.VideoCountry{}, &model.VideoChannel{}, &model.VideoPackage{}, &model.VideoTemplateType{}); err != nil {
		t.Fatal(err)
	}
	position := model.VideoDisplayPosition{PositionName: "Home", PositionKey: "home", CoverImage: "https://example.com/home.jpg", Status: 1}
	cn := model.VideoCountry{Code: "CN", NameZh: "China", Status: 1}
	us := model.VideoCountry{Code: "US", NameZh: "United States", Status: 1}
	channel := model.VideoChannel{ChannelCode: "channel-a", ChannelName: "Channel A", AdPlatform: "direct", DeliveryPackage: "app.a", UploadMethod: "API", Status: 1}
	appPackage := model.VideoPackage{PackageName: "App A", PackageCode: "app.a", PackageVersion: "1.0.0", DownloadURL: "https://example.com/app", Status: 1}
	for _, value := range []interface{}{&position, &cn, &us, &channel, &appPackage} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	types := []model.VideoTemplateType{
		{CategoryName: "Global", UserTypes: []int{1, 2}, SubscriptionStatuses: []string{"subscribed", "unsubscribed"}, Sort: 0, Status: 1},
		{CategoryName: "Matched", UserTypes: []int{1}, SubscriptionStatuses: []string{"unsubscribed"}, Sort: 1, Status: 1},
		{CategoryName: "Wrong country", UserTypes: []int{1, 2}, SubscriptionStatuses: []string{"subscribed", "unsubscribed"}, Sort: 2, Status: 1},
	}
	if err := db.Create(&types).Error; err != nil {
		t.Fatal(err)
	}
	for i := range types {
		if err := db.Model(&types[i]).Association("DisplayPositions").Append(&position); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewTemplateTypeRepo()
	if err := repo.ReplaceTargets(context.Background(), &types[1], TemplateTypeTargetIDs{
		DisplayPositionKeys: []string{"home"}, CountryIDs: []uint64{cn.ID},
		ChannelIDs: []uint64{channel.ChannelID}, PackageIDs: []uint64{appPackage.ID},
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceTargets(context.Background(), &types[2], TemplateTypeTargetIDs{
		DisplayPositionKeys: []string{"home"}, CountryIDs: []uint64{us.ID},
	}); err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListForClient(context.Background(), ClientTemplateTypeTargets{
		PositionKey: "home", CountryID: cn.ID, ChannelIDs: []uint64{channel.ChannelID}, PackageIDs: []uint64{appPackage.ID},
		UserType: 1, SubscriptionState: "unsubscribed",
	})
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(list))
	for i := range list {
		names[i] = list[i].CategoryName
	}
	sort.Strings(names)
	if want := []string{"Global", "Matched"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("template type names = %v, want %v", names, want)
	}
}

func TestTemplatePageListReturnsRowsAndTotal(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-page-list?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.AutoMigrate(
		&model.VideoTemplateType{}, &model.VideoDisplayPosition{}, &model.VideoCountry{},
		&model.VideoPackage{}, &model.VideoChannel{}, &model.VideoTemplate{},
	); err != nil {
		t.Fatal(err)
	}

	typeItem := model.VideoTemplateType{CategoryName: "热门", UserTypes: []int{1, 2}, SubscriptionStatuses: []string{"subscribed", "unsubscribed"}, Status: 1}
	position := model.VideoDisplayPosition{PositionName: "首页", PositionKey: "home", CoverImage: "https://example.com/cover.jpg", Status: 1}
	country := model.VideoCountry{Code: "CN", NameZh: "中国", Status: 1}
	appPackage := model.VideoPackage{PackageName: "示例包", PackageCode: "com.example.app", PackageVersion: "1.0.0", DownloadURL: "https://example.com/app.apk", Status: 1}
	channel := model.VideoChannel{ChannelCode: "META_CN", ChannelName: "Meta 中国", AdPlatform: "Meta Ads", DeliveryPackage: "com.example.app", UploadMethod: "API", Status: 1}
	for _, value := range []interface{}{&typeItem, &position, &country, &appPackage, &channel} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}

	repo := NewTemplateRepo()
	template := model.VideoTemplate{
		VideoTemplateTypeID: typeItem.ID, UserTypes: []int{1, 2},
		SubscriptionStatuses: []string{"subscribed", "unsubscribed"},
		Name:                 "测试模板", TemplateType: model.VideoTemplateKindAction,
		CoverImage: "https://example.com/template.jpg", TemplateVideo: "https://example.com/template.mp4", Status: 1,
	}
	ctx := context.Background()
	if err := repo.Create(ctx, &template); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceTargets(ctx, &template, TemplateTargetIDs{
		CountryIDs: []uint64{country.ID}, PackageIDs: []uint64{appPackage.ID}, ChannelIDs: []uint64{channel.ChannelID},
	}); err != nil {
		t.Fatal(err)
	}

	list, total, err := repo.PageList(ctx, 1, 20, &TemplateListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("total=%d len(list)=%d, want 1 and 1", total, len(list))
	}
	got := list[0]
	if got.ID != template.ID || len(got.Countries) != 1 || len(got.Packages) != 1 || len(got.Channels) != 1 {
		t.Fatalf("template or associations not loaded: %#v", got)
	}
}
