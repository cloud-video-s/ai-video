package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestInitCasbinDoesNotCreatePolicyTable(t *testing.T) {
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	previousDB, previousEnforcer, previousModelPath := DB, Enforcer, Cfg.Casbin.ModelPath
	DB = db
	Cfg.Casbin.ModelPath = filepath.Join("..", "..", "config", "rbac_model.conf")
	t.Cleanup(func() {
		DB, Enforcer, Cfg.Casbin.ModelPath = previousDB, previousEnforcer, previousModelPath
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	if err := InitCasbin(); err == nil {
		t.Fatal("InitCasbin unexpectedly succeeded without casbin_rule")
	}
	if db.Migrator().HasTable("casbin_rule") {
		t.Fatal("InitCasbin created casbin_rule despite automatic migration being disabled")
	}
}
