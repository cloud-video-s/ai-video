package main

import (
	"ai-video/internal/config"
	"flag"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func main() {
	cfgFile := flag.String("config", "", "config file path")
	outPath := flag.String("out", "internal/gen/query", "generated query package path")
	modelPath := flag.String("model", "internal/gen/model", "generated model package path")
	flag.Parse()

	if err := config.InitConfig(*cfgFile); err != nil {
		panic(fmt.Sprintf("init config failed: %v", err))
	}
	if err := config.InitTimezone(); err != nil {
		panic(fmt.Sprintf("init timezone failed: %v", err))
	}

	db, err := openDB()
	if err != nil {
		panic(fmt.Sprintf("connect database failed: %v", err))
	}
	config.DB = db

	g := gen.NewGenerator(gen.Config{
		OutPath:           filepath.FromSlash(*outPath),
		ModelPkgPath:      filepath.FromSlash(*modelPath),
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable:     false,
		FieldCoverable:    false,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
	g.UseDB(db)

	g.GenerateAllTable()
	// 1. casbin_rule
	casbinRule := g.GenerateModel("casbin_rule",
		gen.FieldType("id", "uint64"),
	)

	videoAdminRole := g.GenerateModel("video_admin_role",
		gen.FieldType("id", "uint64"),
		gen.FieldType("video_admin_id", "uint64"),
		gen.FieldType("video_role_id", "uint64"),
	)

	// 15. video_menu
	videoMenu := g.GenerateModel("video_menu",
		gen.FieldType("id", "uint64"),
		gen.FieldType("parent_id", "uint64"),
		gen.FieldType("type", "uint8"),
		gen.FieldType("visible", "uint8"),
		gen.FieldType("status", "uint8"),
		// 自关联：子菜单
		gen.FieldRelate(
			field.HasMany, "ChildMenus", g.GenerateModel("video_menu"),
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"foreignKey": []string{"parent_id"},
					"references": []string{"id"},
				},
			},
		),
	)

	// 24. video_role
	videoRole := g.GenerateModel("video_role",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "uint8"),
		gen.FieldRelate(
			field.Many2Many, "Menus", videoMenu,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_role_menu"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"video_role_id"},
					"joinReferences": []string{"video_menu_id"},
					"References":     []string{"ID"},
				},
			},
		),
	)

	// 2. video_admin
	videoAdmin := g.GenerateModel("video_admin",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("token_version", "int64"),
		gen.FieldRelate(
			field.Many2Many, "Roles", videoRole,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_admin_role"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"video_admin_id"},
					"joinReferences": []string{"video_role_id"},
					"References":     []string{"ID"},
				},
			},
		),
	)

	// 4. video_api
	videoApi := g.GenerateModel("video_api",
		gen.FieldType("id", "uint64"),
	)

	// 5. video_app
	videoApp := g.GenerateModel("video_app",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "uint8"),
		gen.FieldType("sort", "uint"),
	)

	// 6. video_banner
	videoBanner := g.GenerateModel("video_banner",
		gen.FieldType("id", "uint64"),
		gen.FieldType("sort", "uint64"),
		gen.FieldType("jump_type", "uint8"),
		gen.FieldType("status", "int8"),
		gen.FieldType("subscription_status", "uint8"),
	)

	// 7. video_banner_app
	videoBannerApp := g.GenerateModel("video_banner_app",
		gen.FieldType("id", "uint64"),
		gen.FieldType("banner_id", "uint64"),
	)

	// 8. video_banner_country
	videoBannerCountry := g.GenerateModel("video_banner_country",
		gen.FieldType("id", "uint64"),
		gen.FieldType("banner_id", "uint64"),
	)

	// 9. video_banner_display_position
	videoBannerDisplayPos := g.GenerateModel("video_banner_display_position",
		gen.FieldType("id", "uint64"),
		gen.FieldType("banner_id", "uint64"),
	)

	// 10. video_channel
	videoChannel := g.GenerateModel("video_channel",
		gen.FieldType("channel_id", "uint64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("port_rebate", "float64"),
		gen.FieldType("service_order_fee", "float64"),
	)

	// 11. video_config
	videoConfig := g.GenerateModel("video_config",
		gen.FieldType("id", "uint64"),
		gen.FieldType("is_public", "bool"),
		gen.FieldType("editable", "bool"),
		gen.FieldType("builtin", "bool"),
		gen.FieldType("sensitive", "bool"),
	)

	// 12. video_country
	videoCountry := g.GenerateModel("video_country",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "int8"),
	)

	// 13. video_delay_config
	videoDelayConfig := g.GenerateModel("video_delay_config",
		gen.FieldType("id", "uint64"),
	)

	// 14. video_display_position
	videoDisplayPosition := g.GenerateModel("video_display_position",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "int8"),
	)

	// 16. video_menu_api
	videoMenuAPI := g.GenerateModel("video_menu_api",
		gen.FieldType("id", "uint64"),
		gen.FieldType("video_menu_id", "uint64"),
		gen.FieldType("video_api_id", "uint64"),
	)

	// 17. video_operation_log
	videoOperationLog := g.GenerateModel("video_operation_log",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("success", "bool"),
	)

	// 18. video_order
	videoOrder := g.GenerateModel("video_order",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("product_id", "uint64"),
		gen.FieldType("vip_level", "uint"),
		gen.FieldType("vip_duration_days", "uint"),
		gen.FieldType("bonus_points", "uint64"),
	)

	// 19. video_package
	videoPackage := g.GenerateModel("video_package",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "uint8"),
		gen.FieldType("system_type", "uint8"),
	)

	// 20. video_package_version
	videoPackageVersion := g.GenerateModel("video_package_version",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "uint8"),
		gen.FieldType("install_count", "uint64"),
		gen.FieldType("download_count", "uint64"),
		gen.FieldType("device_count", "uint64"),
	)

	// 21. video_points_package
	videoPointsPackage := g.GenerateModel("video_points_package",
		gen.FieldType("id", "uint64"),
		gen.FieldType("points", "uint64"),
		gen.FieldType("is_default", "bool"),
		gen.FieldType("status", "int8"),
		gen.FieldType("sale_price", "float64"),
		gen.FieldType("actual_revenue", "float64"),
		gen.FieldType("original_price", "float64"),
	)

	// 22. video_points_package_channel
	videoPointsPackageChannel := g.GenerateModel("video_points_package_channel",
		gen.FieldType("id", "uint64"),
	)

	// 23. video_points_package_package
	videoPointsPackagePackage := g.GenerateModel("video_points_package_package",
		gen.FieldType("id", "uint64"),
	)

	// 25. video_role_menu
	videoRoleMenu := g.GenerateModel("video_role_menu",
		gen.FieldType("video_role_id", "uint64"),
		gen.FieldType("video_menu_id", "uint64"),
	)

	// 26. video_template
	videoTemplate := g.GenerateModel("video_template",
		gen.FieldType("id", "uint64"),
		gen.FieldType("video_template_type_id", "uint64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("usage_count", "uint64"),
		gen.FieldType("like_count", "uint64"),
		gen.FieldType("view_count", "uint64"),
		gen.FieldType("favorite_count", "uint64"),
		// BelongsTo: video_template_type
		gen.FieldRelate(
			field.BelongsTo, "TemplateType", g.GenerateModel("video_template_type"),
			&field.RelateConfig{
				RelateSlicePointer: false,
				GORMTag: field.GormTag{
					"foreignKey": []string{"video_template_type_id"},
					"references": []string{"id"},
				},
			},
		),
	)

	// 27. video_template_display_config
	videoTemplateDisplayConfig := g.GenerateModel("video_template_display_config",
		gen.FieldType("id", "uint64"),
		gen.FieldType("template_id", "uint64"),
		gen.FieldType("status", "uint8"),
	)

	// 28. video_template_type
	videoTemplateType := g.GenerateModel("video_template_type",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("is_subscribed", "bool"),
		// HasMany: video_template
		gen.FieldRelate(
			field.HasMany, "Templates", videoTemplate,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"foreignKey": []string{"video_template_type_id"},
					"references": []string{"id"},
				},
			},
		),
	)

	// 29. video_template_type_app
	videoTemplateTypeApp := g.GenerateModel("video_template_type_app",
		gen.FieldType("id", "uint64"),
		gen.FieldType("template_type_id", "uint64"),
	)

	// 30. video_template_type_country
	videoTemplateTypeCountry := g.GenerateModel("video_template_type_country",
		gen.FieldType("id", "uint64"),
		gen.FieldType("template_type_id", "uint64"),
	)

	// 31. video_template_type_display_position
	videoTemplateTypeDisplayPos := g.GenerateModel("video_template_type_display_position",
		gen.FieldType("id", "uint64"),
		gen.FieldType("template_type_id", "uint64"),
	)

	// 32. video_upload
	videoUpload := g.GenerateModel("video_upload",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_type", "int8"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("file_size", "uint64"),
	)

	// 33. video_user
	videoUser := g.GenerateModel("video_user",
		gen.FieldType("id", "uint64"),
		gen.FieldType("login_type", "uint8"),
		gen.FieldType("user_type", "uint8"),
		gen.FieldType("active_days", "uint"),
		gen.FieldType("avg_daily_usage_seconds", "uint64"),
		gen.FieldType("points_balance", "uint64"),
		gen.FieldType("subscription_status", "uint8"),
		gen.FieldType("order_count", "uint64"),
		gen.FieldType("payment_count", "uint64"),
		gen.FieldType("subscription_payment_count", "uint64"),
		gen.FieldType("one_time_payment_count", "uint64"),
		gen.FieldType("activated", "uint"),
		gen.FieldType("key_behavior_met", "uint"),
		gen.FieldType("payment_met", "bool"),
		gen.FieldType("first_payment_met", "bool"),
		gen.FieldType("registered", "bool"),
		gen.FieldType("token_version", "int64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("vip_level", "uint"),
		gen.FieldType("is_frozen", "bool"),
		gen.FieldType("is_blacklisted", "bool"),
	)

	// 34. video_user_attribution
	videoUserAttribution := g.GenerateModel("video_user_attribution",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
	)

	// 35. video_user_identity
	videoUserIdentity := g.GenerateModel("video_user_identity",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("email_verified", "bool"),
		gen.FieldType("is_private_email", "bool"),
	)

	// 36. video_user_points_ledger
	videoUserPointsLedger := g.GenerateModel("video_user_points_ledger",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("direction", "int8"),
		gen.FieldType("points_change", "int64"),
		gen.FieldType("balance_before", "uint64"),
		gen.FieldType("balance_after", "uint64"),
		gen.FieldType("points_package_id", "uint64"),
		gen.FieldType("operator_admin_id", "uint64"),
		gen.FieldType("order_id", "uint64"),
	)

	// 37. video_user_template_favorite
	videoUserTemplateFavorite := g.GenerateModel("video_user_template_favorite",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("template_id", "uint64"),
	)

	// 38. video_vip_subscription
	videoVipSubscription := g.GenerateModel("video_vip_subscription",
		gen.FieldType("id", "uint64"),
		gen.FieldType("first_subscription_price", "float64"),
		gen.FieldType("first_subscription_revenue", "float64"),
		gen.FieldType("first_bonus_points", "uint64"),
		gen.FieldType("original_price", "float64"),
		gen.FieldType("v_ip_duration_days", "uint"),
		gen.FieldType("trial_days", "uint"),
		gen.FieldType("agreement_default_checked", "bool"),
		gen.FieldType("display_mode", "int8"),
		gen.FieldType("status", "int8"),
		gen.FieldType("free_trial", "bool"),
		gen.FieldType("is_subscription", "bool"),
		gen.FieldType("is_default", "bool"),
		gen.FieldType("subscription_price", "float64"),
		gen.FieldType("subscription_revenue", "float64"),
		gen.FieldType("subscription_points", "uint64"),
	)

	// 39. video_vip_subscription_channel
	videoVipSubscriptionChannel := g.GenerateModel("video_vip_subscription_channel",
		gen.FieldType("id", "uint64"),
		gen.FieldType("subscription_id", "uint64"),
	)

	// 40. video_vip_subscription_package
	videoVipSubscriptionPackage := g.GenerateModel("video_vip_subscription_package",
		gen.FieldType("id", "uint64"),
		gen.FieldType("subscription_id", "uint64"),
	)

	// 41. video_vip_subscription_position
	videoVipSubscriptionPosition := g.GenerateModel("video_vip_subscription_position",
		gen.FieldType("id", "uint64"),
		gen.FieldType("subscription_id", "uint64"),
	)

	// 42. video_vip_subscription_excluded_channel
	videoVipSubscriptionExcludedChannel := g.GenerateModel("video_vip_subscription_excluded_channel",
		gen.FieldType("subscription_id", "uint64"),
		gen.FieldType("channel_id", "uint64"),
	)

	// ------------------------- 应用并执行生成 -------------------------
	allModels := []interface{}{
		casbinRule,
		videoAdmin,
		videoAdminRole,
		videoApi,
		videoApp,
		videoBanner,
		videoBannerApp,
		videoBannerCountry,
		videoBannerDisplayPos,
		videoChannel,
		videoConfig,
		videoCountry,
		videoDelayConfig,
		videoDisplayPosition,
		videoMenu,
		videoMenuAPI,
		videoOperationLog,
		videoOrder,
		videoPackage,
		videoPackageVersion,
		videoPointsPackage,
		videoPointsPackageChannel,
		videoPointsPackagePackage,
		videoRole,
		videoRoleMenu,
		videoTemplate,
		videoTemplateDisplayConfig,
		videoTemplateType,
		videoTemplateTypeApp,
		videoTemplateTypeCountry,
		videoTemplateTypeDisplayPos,
		videoUpload,
		videoUser,
		videoUserAttribution,
		videoUserIdentity,
		videoUserPointsLedger,
		videoUserTemplateFavorite,
		videoVipSubscription,
		videoVipSubscriptionChannel,
		videoVipSubscriptionPackage,
		videoVipSubscriptionPosition,
		videoVipSubscriptionExcludedChannel,
	}
	g.ApplyBasic(allModels...)

	g.Execute()
}

type tableInfo struct {
	name    string
	columns map[string]struct{}
}

type manyToManyRelation struct {
	owner, target, joinTable          string
	ownerColumn, targetColumn         string
	joinOwnerColumn, joinTargetColumn string
}

func generateModelsWithRelations(g *gen.Generator, db *gorm.DB) ([]interface{}, error) {
	tables, err := inspectTables(db)
	if err != nil {
		return nil, err
	}

	options := make(map[string][]gen.ModelOpt)
	for _, table := range tables {
		options[table.name] = append(options[table.name], businessFieldOptions(table.name)...)
	}
	for _, relation := range discoverManyToMany(tables) {
		// The legacy banner channel column is named channel_code but stores the
		// numeric video_channel.channel_id value.
		if relation.owner == "video_banner" && relation.target == "video_channel" {
			continue
		}
		fieldName := pluralFieldName(relation.target)
		options[relation.owner] = append(options[relation.owner], gen.FieldRelate(
			field.Many2Many,
			fieldName,
			g.GenerateModel(relation.target),
			&field.RelateConfig{
				RelateSlice: true,
				GORMTag: field.GormTag{
					"many2many":      []string{relation.joinTable},
					"foreignKey":     []string{schema.NamingStrategy{}.SchemaName(relation.ownerColumn)},
					"joinForeignKey": []string{schema.NamingStrategy{}.SchemaName(relation.joinOwnerColumn)},
					"joinReferences": []string{schema.NamingStrategy{}.SchemaName(relation.joinTargetColumn)},
					"References":     []string{schema.NamingStrategy{}.SchemaName(relation.targetColumn)},
				},
			},
		))
		fmt.Printf("discovered many2many: %s.%s -> %s via %s\n", relation.owner, fieldName, relation.target, relation.joinTable)
	}
	addBusinessRelations(g, options)
	result := make([]interface{}, 0, len(tables))
	for _, table := range tables {
		result = append(result, g.GenerateModel(table.name, options[table.name]...))
	}
	return result, nil
}

func addBusinessRelations(g *gen.Generator, options map[string][]gen.ModelOpt) {
	addManyToMany := func(owner, name, target, joinTable, joinOwner, joinTarget string) {
		options[owner] = append(options[owner], gen.FieldRelate(field.Many2Many, name, g.GenerateModel(target), &field.RelateConfig{
			RelateSlice: true,
			GORMTag: field.GormTag{
				"many2many": {joinTable}, "foreignKey": {"ID"}, "joinForeignKey": {joinOwner},
				"References": {"ID"}, "joinReferences": {joinTarget},
			},
		}))
	}
	addBelongsTo := func(owner, name, target, foreignKey, reference string, pointer bool) {
		options[owner] = append(options[owner], gen.FieldRelate(field.BelongsTo, name, g.GenerateModel(target), &field.RelateConfig{
			RelatePointer: pointer,
			GORMTag:       field.GormTag{"foreignKey": {foreignKey}, "References": {reference}},
		}))
	}

	addManyToMany("video_admin", "Roles", "video_role", "video_admin_role", "VideoAdminID", "VideoRoleID")
	addManyToMany("video_role", "Menus", "video_menu", "video_role_menu", "VideoRoleID", "VideoMenuID")
	addManyToMany("video_menu", "APIs", "video_api", "video_menu_api", "VideoMenuID", "VideoAPIID")
	options["video_banner"] = append(options["video_banner"], gen.FieldRelate(
		field.Many2Many, "Channels", g.GenerateModel("video_channel"), &field.RelateConfig{
			RelateSlice: true,
			GORMTag: field.GormTag{
				"many2many": {"video_banner_channel"}, "foreignKey": {"ID"},
				"joinForeignKey": {"BannerID"}, "References": {"ChannelID"}, "joinReferences": {"ChannelCode"},
			},
		},
	))
	options["video_points_package"] = append(options["video_points_package"], gen.FieldRelate(
		field.Many2Many, "Channels", g.GenerateModel("video_channel"), &field.RelateConfig{
			RelateSlice: true,
			GORMTag: field.GormTag{
				"many2many": {"video_points_package_channel"}, "foreignKey": {"ProductID"},
				"joinForeignKey": {"ProductCode"}, "References": {"ChannelCode"}, "joinReferences": {"ChannelCode"},
			},
		},
	))
	options["video_vip_subscription"] = append(options["video_vip_subscription"],
		gen.FieldRelate(field.Many2Many, "DisplayPositions", g.GenerateModel("video_display_position"), &field.RelateConfig{
			RelateSlice: true,
			GORMTag: field.GormTag{
				"many2many": {"video_vip_subscription_position"}, "foreignKey": {"ID"},
				"joinForeignKey": {"SubscriptionID"}, "References": {"PositionKey"}, "joinReferences": {"ProductCode"},
			},
		}),
		gen.FieldRelate(field.Many2Many, "ExcludedChannels", g.GenerateModel("video_channel"), &field.RelateConfig{
			RelateSlice: true,
			GORMTag: field.GormTag{
				"many2many": {"video_vip_subscription_excluded_channel"}, "foreignKey": {"ID"},
				"joinForeignKey": {"SubscriptionID"}, "References": {"ChannelID"}, "joinReferences": {"ChannelID"},
			},
		}),
	)

	options["video_menu"] = append(options["video_menu"], gen.FieldRelate(field.HasMany, "Children", g.GenerateModel("video_menu"), &field.RelateConfig{
		RelateSlice: true, GORMTag: field.GormTag{"foreignKey": {"ParentID"}, "References": {"ID"}},
	}))
	addBelongsTo("video_banner", "Template", "video_template", "TemplateID", "ID", true)
	addBelongsTo("video_template", "VideoTemplateType", "video_template_type", "VideoTemplateTypeID", "ID", false)
	addBelongsTo("video_template_display_config", "Template", "video_template", "TemplateID", "ID", false)
	addBelongsTo("video_template_display_config", "DisplayPosition", "video_display_position", "DisplayPositionKey", "PositionKey", false)
	addBelongsTo("video_user_attribution", "User", "video_user", "UserID", "ID", false)
	addBelongsTo("video_user_attribution", "Channel", "video_channel", "ChannelCode", "ChannelCode", true)
	addBelongsTo("video_user_identity", "User", "video_user", "UserID", "ID", false)
	addBelongsTo("video_user_points_ledger", "User", "video_user", "UserID", "ID", false)
	addBelongsTo("video_user_points_ledger", "PointsPackage", "video_points_package", "PointsPackageID", "ID", true)
}

func businessFieldOptions(table string) []gen.ModelOpt {
	jsonSerializer := func(tag field.GormTag) field.GormTag { return tag.Set("serializer", "json") }
	switch table {
	case "video_user":
		return []gen.ModelOpt{
			gen.FieldRename("imei", "IMEI"), gen.FieldRename("appid_email", "AppIDEmail"),
			gen.FieldRename("appid_third_code", "AppIDThirdCode"),
			gen.FieldType("first_opened_at", "*time.Time"), gen.FieldType("last_opened_at", "*time.Time"),
			gen.FieldType("vip_expires_at", "*time.Time"), gen.FieldType("first_order_created_at", "*time.Time"),
			gen.FieldType("first_paid_at", "*time.Time"), gen.FieldType("last_paid_at", "*time.Time"),
			gen.FieldType("attribution_clicked_at", "*time.Time"), gen.FieldType("last_login_at", "*time.Time"),
		}
	case "video_user_attribution":
		return []gen.ModelOpt{
			gen.FieldRename("oaid", "OAID"), gen.FieldRename("imei", "IMEI"),
			gen.FieldType("attributed_at", "*time.Time"), gen.FieldType("last_operated_at", "*time.Time"),
		}
	case "video_template", "video_template_type":
		options := []gen.ModelOpt{
			gen.FieldType("user_types", "[]int"), gen.FieldGORMTag("user_types", jsonSerializer),
			gen.FieldType("subscription_statuses", "[]string"), gen.FieldGORMTag("subscription_statuses", jsonSerializer),
		}
		if table == "video_template" {
			options = append(options, gen.FieldType("sort", "int"), gen.FieldType("status", "int8"))
		} else {
			options = append(options, gen.FieldType("status", "int8"))
		}
		return options
	case "video_upload":
		return []gen.ModelOpt{
			gen.FieldType("user_type", "int8"), gen.FieldRename("mime_type", "MIMEType"),
			gen.FieldRename("sha256", "SHA256"),
		}
	case "video_banner":
		return []gen.ModelOpt{
			gen.FieldType("template_id", "*uint64"), gen.FieldType("jump_type", "uint8"), gen.FieldType("status", "int8"),
		}
	case "video_country", "video_channel":
		return []gen.ModelOpt{gen.FieldType("status", "int8")}
	case "video_display_position", "video_template_display_config":
		options := []gen.ModelOpt{gen.FieldType("sort", "int"), gen.FieldType("status", "int8")}
		if table == "video_template_display_config" {
			options = append(options,
				gen.FieldRename("position_key", "DisplayPositionKey"),
				gen.FieldRename("description", "Remark"), gen.FieldJSONTag("description", "remark"),
			)
		}
		return options
	case "video_package":
		return []gen.ModelOpt{
			gen.FieldType("system_types", "[]string"), gen.FieldGORMTag("system_types", jsonSerializer),
			gen.FieldType("sort", "int"), gen.FieldType("status", "int8"),
		}
	case "video_points_package":
		return []gen.ModelOpt{
			gen.FieldRename("product_code", "ProductID"), gen.FieldJSONTag("product_code", "product_id"),
			gen.FieldType("systems", "[]string"), gen.FieldGORMTag("systems", jsonSerializer),
			gen.FieldType("user_types", "[]int"), gen.FieldGORMTag("user_types", jsonSerializer),
			gen.FieldType("sort", "int"), gen.FieldType("status", "int8"),
		}
	case "video_vip_subscription":
		return []gen.ModelOpt{
			gen.FieldJSONTag("v_ip_level", "vip_level"), gen.FieldJSONTag("v_ip_duration_days", "vip_duration_days"),
			gen.FieldType("display_mode", "int8"), gen.FieldType("status", "int8"), gen.FieldType("sort", "int"),
		}
	case "video_user_identity":
		return []gen.ModelOpt{
			gen.FieldType("last_login_at", "*time.Time"), gen.FieldType("last_token_issued_at", "*time.Time"),
		}
	case "video_user_points_ledger":
		return []gen.ModelOpt{
			gen.FieldType("points_package_id", "*uint64"), gen.FieldType("operator_admin_id", "*uint64"),
		}
	default:
		return nil
	}
}

func inspectTables(db *gorm.DB) ([]tableInfo, error) {
	names, err := db.Migrator().GetTables()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	tables := make([]tableInfo, 0, len(names))
	for _, name := range names {
		columnTypes, err := db.Migrator().ColumnTypes(name)
		if err != nil {
			return nil, fmt.Errorf("read columns of %s: %w", name, err)
		}
		columns := make(map[string]struct{}, len(columnTypes))
		for _, column := range columnTypes {
			columns[strings.ToLower(column.Name())] = struct{}{}
		}
		tables = append(tables, tableInfo{name: name, columns: columns})
	}
	return tables, nil
}

func discoverManyToMany(tables []tableInfo) []manyToManyRelation {
	byName := make(map[string]tableInfo, len(tables))
	for _, table := range tables {
		byName[table.name] = table
	}

	var relations []manyToManyRelation
	for _, join := range tables {
		for _, owner := range tables {
			prefix := owner.name + "_"
			if !strings.HasPrefix(join.name, prefix) {
				continue
			}
			remainder := strings.TrimPrefix(join.name, prefix)
			target, ok := byName[remainder]
			if !ok {
				target, ok = byName["video_"+remainder]
			}
			if !ok || target.name == owner.name {
				continue
			}
			ownerJoin, ownerRef, okOwner := relationColumns(owner, join)
			targetJoin, targetRef, okTarget := relationColumns(target, join)
			if !okOwner || !okTarget || ownerJoin == targetJoin {
				continue
			}
			relations = append(relations, manyToManyRelation{
				owner: owner.name, target: target.name, joinTable: join.name,
				ownerColumn: ownerRef, targetColumn: targetRef,
				joinOwnerColumn: ownerJoin, joinTargetColumn: targetJoin,
			})
			break // the longest table prefix wins because tables are sorted
		}
	}
	return relations
}

func relationColumns(base, join tableInfo) (joinColumn, referenceColumn string, ok bool) {
	short := strings.TrimPrefix(base.name, "video_")
	candidates := []string{short + "_id", short + "_code", short + "_key"}
	// display_position commonly uses position_key in association tables.
	if strings.Contains(short, "_") {
		last := short[strings.LastIndex(short, "_")+1:]
		candidates = append(candidates, last+"_id", last+"_code", last+"_key")
	}
	for _, candidate := range candidates {
		if _, exists := join.columns[candidate]; !exists {
			continue
		}
		suffix := candidate[strings.LastIndex(candidate, "_")+1:]
		ref := suffix
		if suffix == "id" {
			ref = "id"
		} else if _, exists := base.columns[short+"_"+suffix]; exists {
			ref = short + "_" + suffix
		} else if _, exists := base.columns[candidate]; exists {
			ref = candidate
		} else if _, exists := base.columns[suffix]; !exists {
			continue
		}
		if _, exists := base.columns[ref]; exists {
			return candidate, ref, true
		}
	}
	return "", "", false
}

func pluralFieldName(table string) string {
	name := schema.NamingStrategy{}.SchemaName(table)
	name = strings.TrimPrefix(name, "Video")
	if strings.HasSuffix(name, "y") && !strings.HasSuffix(name, "ay") {
		return strings.TrimSuffix(name, "y") + "ies"
	}
	if strings.HasSuffix(name, "s") {
		return name
	}
	return name + "s"
}

func openDB() (*gorm.DB, error) {
	cfg := config.Cfg.Database
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.DSN(config.Cfg.Timezone))
	case "mysql":
		dialector = mysql.Open(cfg.DSN(config.Cfg.Timezone))
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	return db, nil
}
