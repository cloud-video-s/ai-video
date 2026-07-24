package main

import (
	"ai-video/internal/config"
	"flag"
	"fmt"
	"path/filepath"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	videoAdminRole := g.GenerateModel("video_admin_role")
	casbinRule := g.GenerateModel("casbin_rule",
		gen.FieldType("id", "uint64"),
	)
	videoRoleMenu := g.GenerateModel("video_role_menu")
	videoApp := g.GenerateModel("video_app",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "uint8"),
		gen.FieldType("sort", "uint"),
	)
	videoPackage := g.GenerateModel("video_package",
		gen.FieldType("id", "uint64"),
		gen.FieldType("sort", "int64"),
		gen.FieldType("status", "uint8"),
		gen.FieldType("system_type", "uint8"),
		gen.FieldRelate(field.BelongsTo, "App", videoApp,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"AppCode"},
					"references": []string{"AppCode"},
				},
			},
		),
	)
	videoPackageVersion := g.GenerateModel("video_package_version",
		gen.FieldType("id", "uint64"),
		gen.FieldType("install_count", "uint64"),
		gen.FieldType("download_count", "uint64"),
		gen.FieldType("device_count", "uint64"),
		gen.FieldType("status", "uint8"),
		gen.FieldRelate(field.BelongsTo, "Package", videoPackage,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"PackageCode"},
					"references": []string{"PackageCode"},
				},
			},
		),
	)
	videoCountry := g.GenerateModel("video_country",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "int8"),
	)

	videoMenuAPI := g.GenerateModel("video_menu_api",
		gen.FieldType("id", "uint64"),
		gen.FieldType("video_menu_id", "uint64"),
		gen.FieldType("video_api_id", "uint64"),
	)

	videoMenu := g.GenerateModel("video_menu",
		gen.FieldType("id", "uint64"),
		gen.FieldType("parent_id", "uint64"),
		gen.FieldType("sort", "uint64"),
		gen.FieldType("type", "uint8"),
		gen.FieldType("visible", "uint8"),
		gen.FieldType("status", "uint8"),
		// 父菜单 (BelongsTo)
		gen.FieldRelate(field.BelongsTo, "ParentMenu", g.GenerateModel("video_menu"),
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"ParentID"},
					"references": []string{"ID"},
				},
			},
		),
		// 子菜单 (HasMany)
		gen.FieldRelate(field.HasMany, "ChildMenus", g.GenerateModel("video_menu"),
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"foreignKey": []string{"ParentID"},
					"references": []string{"ID"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "APIs", g.GenerateModel("video_api"),
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_menu_api"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"VideoMenuID"},
					"joinReferences": []string{"VideoApiID"},
					"References":     []string{"ID"},
				},
			},
		),
	)

	videoRole := g.GenerateModel("video_role",
		gen.FieldType("id", "uint64"),
		gen.FieldType("sort", "int64"),
		gen.FieldType("status", "uint8"),
		gen.FieldRelate(field.Many2Many, "Menus", videoMenu,
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
	videoAdmin := g.GenerateModel("video_admin",
		gen.FieldType("id", "uint64"),
		gen.FieldType("status", "uint8"),
		gen.FieldType("token_version", "int64"),
		// Many2Many: 通过 video_admin_role 关联到 video_role
		gen.FieldRelate(field.Many2Many, "Roles", videoRole,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_admin_role"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"VideoAdminId"},
					"joinReferences": []string{"VideoRoleId"},
					"References":     []string{"ID"},
				},
			},
		),
	)

	videoAPI := g.GenerateModel("video_api",
		gen.FieldType("id", "uint64"),
	)

	videoBannerApp := g.GenerateModel("video_banner_app")

	videoBannerPackage := g.GenerateModel("video_banner_package")
	videoBannerVersion := g.GenerateModel("video_banner_version")

	videoBannerCountry := g.GenerateModel("video_banner_country",
		gen.FieldType("id", "uint64"),
		gen.FieldType("banner_id", "uint64"),
	)

	videoBannerPlacementAssociation := g.GenerateModel("video_banner_placement_association",
		gen.FieldType("id", "uint64"),
		gen.FieldType("banner_id", "uint64"),
	)

	videoBanner := g.GenerateModel("video_banner",
		gen.FieldType("id", "uint64"),
		gen.FieldType("sort", "uint64"),
		gen.FieldType("jump_type", "uint8"),
		gen.FieldType("template_id", "*uint64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("subscription_status", "uint8"),
		gen.FieldRelate(field.BelongsTo, "Template", g.GenerateModel("video_template"),
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"TemplateID"},
					"references": []string{"ID"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Placement", videoBannerPlacementAssociation,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_banner_placement_association"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"BannerID"},
					"joinReferences": []string{"PlacementKey"},
					"References":     []string{"PlacementKey"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Countrys", videoCountry,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_banner_country"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"BannerID"},
					"joinReferences": []string{"CountryCode"},
					"References":     []string{"Code"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "App", videoApp,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_banner_app"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"BannerID"},
					"joinReferences": []string{"AppCode"},
					"References":     []string{"AppCode"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Package", videoPackage,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_banner_package"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"BannerID"},
					"joinReferences": []string{"PackageCode"},
					"References":     []string{"PackageCode"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Version", videoPackageVersion,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_banner_version"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"BannerID"},
					"joinReferences": []string{"VersionCode"},
					"References":     []string{"VersionCode"},
				},
			},
		),
	)

	videoChannel := g.GenerateModel("video_channel",
		gen.FieldType("channel_id", "uint64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("port_rebate", "float64"),
		gen.FieldType("service_order_fee", "float64"),
	)

	videoConfig := g.GenerateModel("video_config",
		gen.FieldType("id", "uint64"),
		gen.FieldType("is_public", "int8"),
		gen.FieldType("editable", "int8"),
		gen.FieldType("builtin", "int8"),
		gen.FieldType("sort", "int64"),
		gen.FieldType("sensitive", "int8"),
	)

	videoDelayConfig := g.GenerateModel("video_delay_config",
		gen.FieldType("id", "uint64"),
		gen.FieldType("sort", "int64"),
	)

	videoDisplayPosition := g.GenerateModel("video_display_position",
		gen.FieldType("id", "uint64"),
		gen.FieldType("sort", "int64"),
		gen.FieldType("status", "int8"),
	)

	videoOperationLog := g.GenerateModel("video_operation_log",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("status", "int64"),
		gen.FieldType("biz_code", "int64"),
		gen.FieldType("success", "int8"),
		gen.FieldType("latency_ms", "int64"),
	)

	videoOrder := g.GenerateModel("video_order",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("product_id", "uint64"),
		gen.FieldType("product_amount", "float64"),
		gen.FieldType("discount_amount", "float64"),
		gen.FieldType("payable_amount", "float64"),
		gen.FieldType("paid_amount", "float64"),
		gen.FieldType("refunded_amount", "float64"),
		gen.FieldType("bonus_points", "uint64"),
		gen.FieldType("vip_level", "uint"),
		gen.FieldType("vip_duration_days", "uint"),
		gen.FieldRelate(field.BelongsTo, "User", g.GenerateModel("video_user"),
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"UserID"},
					"references": []string{"ID"},
				},
			},
		),
	)

	videoPointsPackage := g.GenerateModel("video_points_package",
		gen.FieldType("id", "uint64"),
		gen.FieldType("points", "uint64"),
		gen.FieldType("sale_price", "float64"),
		gen.FieldType("actual_revenue", "float64"),
		gen.FieldType("original_price", "float64"),
		gen.FieldType("is_default", "int8"),
		gen.FieldType("status", "int8"),
		gen.FieldType("sort", "int64"),
	)

	videoPointsPackageChannel := g.GenerateModel("video_points_package_channel",
		gen.FieldType("id", "uint64"),
		gen.FieldRelate(field.BelongsTo, "PointsPackage", videoPointsPackage,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"ProductCode"},
					"references": []string{"ProductCode"},
				},
			},
		),
	)

	videoPointsPackagePackage := g.GenerateModel("video_points_package_package",
		gen.FieldType("id", "uint64"),
		gen.FieldRelate(field.BelongsTo, "PointsPackage", videoPointsPackage,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"ProductCode"},
					"references": []string{"ProductCode"},
				},
			},
		),
	)

	videoTemplatePlacement := g.GenerateModel("video_template_placement")

	videoTemplate := g.GenerateModel("video_template",
		gen.FieldType("id", "uint64"),
		gen.FieldType("video_template_type_id", "uint64"),
		gen.FieldType("sort", "int64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("usage_count", "uint64"),
		gen.FieldType("like_count", "uint64"),
		gen.FieldType("view_count", "uint64"),
		gen.FieldType("favorite_count", "uint64"),
		gen.FieldRelate(field.BelongsTo, "VideoTemplateType", g.GenerateModel("video_template_type"),
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"VideoTemplateTypeID"},
					"references": []string{"ID"},
				},
			},
		),
	)

	videoTemplatePlacementConfig := g.GenerateModel("video_template_placement_config",
		gen.FieldType("id", "uint64"),
		gen.FieldType("template_id", "uint64"),
		gen.FieldType("sort", "uint"),
		gen.FieldType("status", "uint8"),
		gen.FieldRelate(field.BelongsTo, "Template", videoTemplate,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"TemplateID"},
					"references": []string{"ID"},
				},
			},
		),
		gen.FieldRelate(field.BelongsTo, "Placement", videoTemplatePlacement,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"PlacementKey"},
					"references": []string{"PlacementKey"},
				},
			},
		),
	)

	videoTemplateType := g.GenerateModel("video_template_type",
		gen.FieldType("id", "uint64"),
		gen.FieldType("sort", "int64"),
		gen.FieldType("status", "int8"),
		gen.FieldRelate(field.Many2Many, "DisplayPosition", videoDisplayPosition,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_template_type_display_position"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"TemplateTypeID"},
					"joinReferences": []string{"PositionKey"},
					"References":     []string{"PositionKey"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Countrys", videoCountry,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_template_type_country"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"TemplateTypeID"},
					"joinReferences": []string{"CountryCode"},
					"References":     []string{"Code"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "App", videoApp,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_template_type_app"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"TemplateTypeID"},
					"joinReferences": []string{"AppCode"},
					"References":     []string{"AppCode"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Package", videoPackage,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_template_type_package"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"TemplateTypeID"},
					"joinReferences": []string{"PackageCode"},
					"References":     []string{"PackageCode"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Version", videoPackageVersion,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_template_type_version"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"TemplateTypeID"},
					"joinReferences": []string{"VersionCode"},
					"References":     []string{"VersionCode"},
				},
			},
		),
	)

	videoTemplateTypeApp := g.GenerateModel("video_template_type_app")
	videoTemplateTypePackage := g.GenerateModel("video_template_type_package")
	videoTemplateTypeVersion := g.GenerateModel("video_template_type_version")

	videoTemplateTypeCountry := g.GenerateModel("video_template_type_country",
		gen.FieldType("id", "uint64"),
		gen.FieldType("template_type_id", "uint64"),
	)

	videoTemplateTypeDisplayPosition := g.GenerateModel("video_template_type_display_position",
		gen.FieldType("id", "uint64"),
		gen.FieldType("template_type_id", "uint64"),
	)

	videoUpload := g.GenerateModel("video_upload",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_type", "int8"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("file_size", "uint64"),
		gen.FieldRename("mime_type", "MIMEType"),
		gen.FieldRename("sha256", "SHA256"),
	)

	// AI 模型配置中心。APIKey 只允许通过管理服务的脱敏视图返回。
	videoAIModel := g.GenerateModel("video_ai_model",
		gen.FieldType("id", "uint64"),
		gen.FieldType("http_timeout_seconds", "uint32"),
		gen.FieldType("poll_interval_seconds", "uint32"),
		gen.FieldType("task_timeout_seconds", "uint32"),
		gen.FieldType("status", "int8"),
		gen.FieldJSONTag("api_key", "-"),
	)

	// 客户端用户生成任务。可空时间使用指针区分“尚未发生”和零值。
	videoGenerationTask := g.GenerateModel("video_generation_task")

	// ================================================================
	// 33. video_user
	// ================================================================
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
		gen.FieldType("order_amount_money", "float64"),
		gen.FieldType("actual_amount_money", "float64"),
		gen.FieldType("refund_amount_money", "float64"),
		gen.FieldType("points_money", "uint64"),
		gen.FieldType("ai_cots_money", "float64"),
		gen.FieldType("activated", "uint"),
		gen.FieldType("key_behavior_met", "uint"),
		gen.FieldType("payment_met", "int8"),
		gen.FieldType("first_payment_met", "int8"),
		gen.FieldType("registered", "int8"),
		gen.FieldType("token_version", "int64"),
		gen.FieldType("status", "int8"),
		gen.FieldType("vip_level", "uint"),
		gen.FieldType("is_frozen", "int8"),
		gen.FieldType("is_blacklisted", "int8"),
		gen.FieldType("first_opened_at", "*time.Time"),
		gen.FieldType("last_opened_at", "*time.Time"),
		gen.FieldType("vip_expires_at", "*time.Time"),
		gen.FieldType("first_order_created_at", "*time.Time"),
		gen.FieldType("first_paid_at", "*time.Time"),
		gen.FieldType("last_paid_at", "*time.Time"),
		gen.FieldType("attribution_clicked_at", "*time.Time"),
		gen.FieldType("last_login_at", "*time.Time"),
		gen.FieldType("vip_started_at", "*time.Time"),
		gen.FieldRename("imei", "IMEI"),
		gen.FieldRename("vip_level", "VIPLevel"),
		gen.FieldRename("vip_started_at", "VIPStartedAt"),
		gen.FieldRelate(field.BelongsTo, "Channel", videoChannel,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"ChannelID"},
					"references": []string{"ChannelCode"},
				},
			},
		),
	)

	// ================================================================
	// 34. video_user_attribution
	// ================================================================
	videoUserAttribution := g.GenerateModel("video_user_attribution",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("activation_callback_count", "uint64"),
		gen.FieldType("activation_deduct_count", "uint64"),
		gen.FieldType("key_behavior_callback_count", "uint64"),
		gen.FieldType("key_behavior_deduct_count", "uint64"),
		gen.FieldType("payment_callback_count", "uint64"),
		gen.FieldType("payment_deduct_count", "uint64"),
		gen.FieldType("first_payment_callback_count", "uint64"),
		gen.FieldType("first_payment_deduct_count", "uint64"),
		gen.FieldType("registration_callback_count", "uint64"),
		gen.FieldType("registration_deduct_count", "uint64"),
		gen.FieldType("attributed_at", "*time.Time"),
		gen.FieldType("last_operated_at", "*time.Time"),
		gen.FieldRename("oaid", "OAID"),
		gen.FieldRename("imei", "IMEI"),
		gen.FieldRelate(field.BelongsTo, "User", videoUser,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"UserID"},
					"references": []string{"ID"},
				},
			},
		),
		gen.FieldRelate(field.BelongsTo, "Channel", videoChannel,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"ChannelCode"},
					"references": []string{"ChannelCode"},
				},
			},
		),
	)

	// ================================================================
	// 35. video_user_identity
	// ================================================================
	videoUserIdentity := g.GenerateModel("video_user_identity",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("email_verified", "int8"),
		gen.FieldType("is_private_email", "int8"),
		gen.FieldType("last_login_at", "*time.Time"),
		gen.FieldType("last_token_issued_at", "*time.Time"),
		gen.FieldRelate(field.BelongsTo, "User", videoUser,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"UserID"},
					"references": []string{"ID"},
				},
			},
		),
	)

	// ================================================================
	// 36. video_user_points_ledger
	// ================================================================
	videoUserPointsLedger := g.GenerateModel("video_user_points_ledger",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("direction", "int8"),
		gen.FieldType("points_change", "int64"),
		gen.FieldType("balance_before", "uint64"),
		gen.FieldType("balance_after", "uint64"),
		gen.FieldType("points_package_id", "*uint64"),
		gen.FieldType("operator_admin_id", "*uint64"),
		gen.FieldType("order_id", "uint64"),
		gen.FieldRelate(field.BelongsTo, "User", videoUser,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"UserID"},
					"references": []string{"ID"},
				},
			},
		),
		gen.FieldRelate(field.BelongsTo, "PointsPackage", videoPointsPackage,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"PointsPackageID"},
					"references": []string{"ID"},
				},
			},
		),
	)

	// ================================================================
	// 37. video_user_template_favorite
	// ================================================================
	videoUserTemplateFavorite := g.GenerateModel("video_user_template_favorite",
		gen.FieldType("id", "uint64"),
		gen.FieldType("user_id", "uint64"),
		gen.FieldType("template_id", "uint64"),
		gen.FieldRelate(field.BelongsTo, "User", videoUser,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"UserID"},
					"references": []string{"ID"},
				},
			},
		),
		gen.FieldRelate(field.BelongsTo, "Template", videoTemplate,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"TemplateID"},
					"references": []string{"ID"},
				},
			},
		),
	)

	videoVipPlacement := g.GenerateModel("video_vip_placement")
	videoVipSubscriptionLevel := g.GenerateModel("video_vip_subscription_level")
	videoVipSubscriptionApp := g.GenerateModel("video_vip_subscription_app")
	videoVipSubscriptionPackage := g.GenerateModel("video_vip_subscription_package")
	videoVipSubscriptionVersion := g.GenerateModel("video_vip_subscription_version")
	videoVipSubscriptionCountry := g.GenerateModel("video_vip_subscription_country")
	videoVipSubscriptionChannel := g.GenerateModel("video_vip_subscription_channel")
	videoVipSubscription := g.GenerateModel("video_vip_subscription",
		gen.FieldType("id", "uint64"),
		gen.FieldType("first_subscription_price", "float64"),
		gen.FieldType("first_subscription_revenue", "float64"),
		gen.FieldType("first_bonus_points", "uint64"),
		gen.FieldType("original_price", "float64"),
		gen.FieldType("v_ip_duration_days", "uint"),
		gen.FieldType("trial_days", "uint"),
		gen.FieldType("agreement_default_checked", "int8"),
		gen.FieldType("display_mode", "int8"),
		gen.FieldType("status", "int8"),
		gen.FieldType("free_trial", "int8"),
		gen.FieldType("is_subscription", "int8"),
		gen.FieldType("is_default", "int8"),
		gen.FieldType("subscription_price", "float64"),
		gen.FieldType("subscription_revenue", "float64"),
		gen.FieldType("subscription_points", "uint64"),
		gen.FieldType("sort", "int64"),
		gen.FieldJSONTag("v_ip_level", "vip_level"),
		gen.FieldJSONTag("v_ip_duration_days", "vip_duration_days"),
		gen.FieldRelate(field.BelongsTo, "SubscriptionLevel", videoVipSubscriptionLevel,
			&field.RelateConfig{
				GORMTag: field.GormTag{
					"foreignKey": []string{"LevelId"},
					"references": []string{"ID"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Apps", videoApp,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_vip_subscription_app"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"SubscriptionID"},
					"joinReferences": []string{"AppCode"},
					"References":     []string{"AppCode"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Packages", videoPackage,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_vip_subscription_package"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"SubscriptionID"},
					"joinReferences": []string{"PackageCode"},
					"References":     []string{"PackageCode"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "PackageVersion", videoPackageVersion,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_vip_subscription_version"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"SubscriptionID"},
					"joinReferences": []string{"VersionCode"},
					"References":     []string{"VersionCode"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Country", videoCountry,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_vip_subscription_country"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"SubscriptionID"},
					"joinReferences": []string{"CountryCode"},
					"References":     []string{"Code"},
				},
			},
		),
		gen.FieldRelate(field.Many2Many, "Channels", videoChannel,
			&field.RelateConfig{
				RelateSlicePointer: true,
				GORMTag: field.GormTag{
					"many2many":      []string{"video_vip_subscription_channel"},
					"foreignKey":     []string{"ID"},
					"joinForeignKey": []string{"SubscriptionID"},
					"joinReferences": []string{"ChannelCode"},
					"References":     []string{"ChannelCode"},
				},
			},
		),
	)

	allModels := []interface{}{
		casbinRule,
		videoAdmin,
		videoAdminRole,
		videoAPI,
		videoApp,
		videoBanner,
		videoBannerApp,
		videoBannerPackage,
		videoBannerVersion,
		videoBannerPlacementAssociation,
		videoBannerCountry,
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
		videoTemplatePlacement,
		videoTemplatePlacementConfig,
		videoTemplateType,
		videoTemplateTypeApp,
		videoTemplateTypePackage,
		videoTemplateTypeVersion,
		videoTemplateTypeCountry,
		videoTemplateTypeDisplayPosition,
		videoUpload,
		videoAIModel,
		videoGenerationTask,
		videoUser,
		videoUserAttribution,
		videoUserIdentity,
		videoUserPointsLedger,
		videoUserTemplateFavorite,
		videoVipSubscription,
		videoVipPlacement,
		videoVipSubscriptionLevel,
		videoVipSubscriptionApp,
		videoVipSubscriptionPackage,
		videoVipSubscriptionVersion,
		videoVipSubscriptionCountry,
		videoVipSubscriptionChannel,
	}

	g.ApplyBasic(allModels...)
	g.Execute()
}

func openDB() (*gorm.DB, error) {
	cfg := config.Cfg.Database
	dialector := mysql.Open(cfg.DSN(config.Cfg.Timezone))
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

func jsonSerializer(tag field.GormTag) field.GormTag {
	return tag.Set("serializer", "json")
}
