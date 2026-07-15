package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"ai-video/internal/app"
	bizmodel "ai-video/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
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
		if err := db.AutoMigrate(
			&bizmodel.VideoDelayConfig{},
			&bizmodel.VideoUser{},
		); err != nil {
			panic(fmt.Sprintf("migrate generated models failed: %v", err))
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
