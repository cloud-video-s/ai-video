package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"ai-video/internal/app"
	bizmodel "ai-video/internal/model"
	"ai-video/internal/repository"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfgFile := flag.String("config", "", "config file path")
	outPath := flag.String("out", "internal/gen/query", "generated query package path")
	modelPath := flag.String("model", "internal/gen/model", "generated model package path")
	source := flag.String("source", "db", "generate from db tables")
	migrate := flag.Bool("migrate", false, "run AutoMigrate before generating")
	flag.Parse()

	if err := app.InitConfig(*cfgFile); err != nil {
		panic(fmt.Sprintf("init config failed: %v", err))
	}
	if err := app.InitTimezone(); err != nil {
		panic(fmt.Sprintf("init timezone failed: %v", err))
	}

	db, err := openDB()
	if err != nil {
		panic(fmt.Sprintf("connect database failed: %v", err))
	}
	app.DB = db

	if *migrate {
		if err := app.PrepareVideoUserColumns(db); err != nil {
			panic(fmt.Sprintf("prepare video user columns failed: %v", err))
		}
		if err := app.NormalizeUserAttributionColumns(db); err != nil {
			panic(fmt.Sprintf("normalize attribution columns failed: %v", err))
		}
		if err := app.MigrateLegacyUploadOwnerColumns(db); err != nil {
			panic(fmt.Sprintf("migrate upload owner columns failed: %v", err))
		}
		if db.Migrator().HasTable("video_app_user") && !db.Migrator().HasTable(&bizmodel.VideoUser{}) {
			if err := db.Migrator().RenameTable("video_app_user", bizmodel.VideoUser{}.TableName()); err != nil {
				panic(fmt.Sprintf("rename client user table failed: %v", err))
			}
		}
		if db.Migrator().HasTable(&bizmodel.VideoUser{}) &&
			db.Migrator().HasColumn("video_user", "v_ip_expires_at") &&
			!db.Migrator().HasColumn("video_user", "vip_expires_at") {
			if err := db.Migrator().RenameColumn("video_user", "v_ip_expires_at", "vip_expires_at"); err != nil {
				panic(fmt.Sprintf("rename video app user VIP column failed: %v", err))
			}
		}
		if err := app.MigrateTemplateTypeDisplayPositionKeys(db); err != nil {
			panic(fmt.Sprintf("migrate template type position keys failed: %v", err))
		}
		if err := db.AutoMigrate(
			&bizmodel.VideoDelayConfig{},
			&bizmodel.VideoTemplateType{},
			&bizmodel.VideoTemplateTypeDisplayPosition{},
			&bizmodel.VideoDisplayPosition{},
			&bizmodel.VideoTemplate{},
			&bizmodel.VideoPackage{},
			&bizmodel.VideoVIPSubscription{},
			&bizmodel.VideoBanner{},
			&bizmodel.VideoUser{},
			&bizmodel.VideoUserIdentity{},
			&bizmodel.VideoConfig{},
			&bizmodel.VideoUpload{},
			&bizmodel.VideoUserAttribution{},
		); err != nil {
			panic(fmt.Sprintf("migrate generated models failed: %v", err))
		}
		if err := app.DropDeprecatedVideoUserColumns(db); err != nil {
			panic(fmt.Sprintf("drop deprecated video user columns failed: %v", err))
		}
		if err := app.MigrateLegacyBannerPositions(db); err != nil {
			panic(fmt.Sprintf("migrate legacy banner positions failed: %v", err))
		}
		if err := app.MigrateLegacyTemplateTypeTargets(db); err != nil {
			panic(fmt.Sprintf("migrate legacy template type targets failed: %v", err))
		}
		if err := app.RemoveLegacyTemplateTypeColumns(db); err != nil {
			panic(fmt.Sprintf("remove legacy template type columns failed: %v", err))
		}
		indexes, err := db.Migrator().GetIndexes(&bizmodel.VideoUser{})
		if err != nil {
			panic(fmt.Sprintf("read client user indexes failed: %v", err))
		}
		for _, index := range indexes {
			name := index.Name()
			if strings.HasPrefix(name, "idx_video_app_user_") ||
				name == "idx_video_user_v_ip_expires_at" ||
				name == "idx_video_user_phone_brand" ||
				name == "idx_video_user_phone_registration" {
				if err := db.Migrator().DropIndex(&bizmodel.VideoUser{}, name); err != nil {
					panic(fmt.Sprintf("drop obsolete client user index %s failed: %v", name, err))
				}
			}
		}
		if err := app.SeedOBDelayConfig(app.DefaultOBDelayConfigPath); err != nil {
			panic(fmt.Sprintf("seed ob delay config failed: %v", err))
		}
		if err := app.SeedDelayConfigAdmin(); err != nil {
			panic(fmt.Sprintf("seed delay config admin failed: %v", err))
		}
		if err := app.SeedAppUserAdmin(); err != nil {
			panic(fmt.Sprintf("seed app user admin failed: %v", err))
		}
		if err := app.SeedUserAttributionAdmin(); err != nil {
			panic(fmt.Sprintf("seed user attribution admin failed: %v", err))
		}
		if _, err := repository.NewUserAttributionRepo().SyncUsers(context.Background()); err != nil {
			panic(fmt.Sprintf("sync user attributions failed: %v", err))
		}
		if err := app.SeedUploadAdmin(); err != nil {
			panic(fmt.Sprintf("seed upload admin failed: %v", err))
		}
		if err := app.SeedTemplateAdmin(); err != nil {
			panic(fmt.Sprintf("seed template admin failed: %v", err))
		}
		if err := app.SeedDisplayPositionAdmin(); err != nil {
			panic(fmt.Sprintf("seed display position admin failed: %v", err))
		}
		if err := app.SeedPackageAdmin(); err != nil {
			panic(fmt.Sprintf("seed package admin failed: %v", err))
		}
		if err := app.SeedVIPSubscriptionAdmin(); err != nil {
			panic(fmt.Sprintf("seed VIP subscription admin failed: %v", err))
		}
		if err := app.SeedBannerAdmin(); err != nil {
			panic(fmt.Sprintf("seed banner admin failed: %v", err))
		}
	}

	g := gen.NewGenerator(gen.Config{
		OutPath:           filepath.FromSlash(*outPath),
		ModelPkgPath:      filepath.FromSlash(*modelPath),
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
	g.UseDB(db)
	switch *source {
	case "db":
		g.ApplyBasic(g.GenerateAllTable()...)
	default:
		panic(fmt.Sprintf("unsupported source: %s", *source))
	}

	g.Execute()
}

func openDB() (*gorm.DB, error) {
	cfg := app.Cfg.Database
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.DSN(app.Cfg.Timezone))
	case "mysql":
		dialector = mysql.Open(cfg.DSN(app.Cfg.Timezone))
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

func dbInit() {
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./internal/gen/query", // 生成代码的输出目录
		ModelPkgPath: "./internal/gen/model",
		Mode:         gen.WithDefaultQuery | gen.WithQueryInterface, // 关键：生成默认查询和接口
		// 自定义生成的结构体字段类型
		FieldNullable:     true, // 字段可为空
		FieldCoverable:    true, // 字段可覆盖
		FieldSignable:     true, // 字段符号
		FieldWithIndexTag: true, // 生成索引标签
		FieldWithTypeTag:  true, // 生成类型标签
	})

	g.UseDB(app.DB)

	g.GenerateAllTable()

	casbinRuleModel := g.GenerateModel("casbin_rule")

	roleModel := g.GenerateModel("video_role")

	adminRoleModel := g.GenerateModel("video_admin_role", gen.FieldRelate(
		field.Many2Many, "Role", roleModel,
		&field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: field.GormTag{"many2many": []string{"video_admin_role"},
				"foreignKey":     []string{"ID"},
				"joinForeignKey": []string{"RoleID"},
				"joinReferences": []string{"ID"},
				"References":     []string{"ID"},
			},
		},
	))

	adminModel := g.GenerateModel("video_admin", gen.FieldRelate(
		field.Many2Many, "Menu", adminRoleModel,
		&field.RelateConfig{
			RelateSlicePointer: true,
			GORMTag: field.GormTag{
				"many2many":      []string{"video_admin_role"},
				"foreignKey":     []string{"ID"},
				"joinForeignKey": []string{"ID"},
				"joinReferences": []string{"RoleID"},
				"References":     []string{"ID"},
			},
		},
	))

	//// 生成基础查询代码（关键步骤）
	//g.ApplyBasic(
	//	adminModel,
	//	postModel,
	//	// 可以添加更多表...
	//)
	//
	//// 生成关联查询代码（如果需要）
	//g.ApplyInterface(func(Interface) {}, userModel, postModel)

	// 生成关联查询代码
	g.ApplyBasic(
		casbinRuleModel,
		adminRoleModel,
		adminModel,
	)

	// 执行生成
	g.Execute()
}
