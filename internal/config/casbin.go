package config

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

var Enforcer *casbin.SyncedEnforcer

func InitCasbin() error {
	// The adapter enables AutoMigrate by default. Production services must never
	// create or alter casbin_rule during startup; schema is managed exclusively
	// by reviewed scripts under scripts/schema.
	gormadapter.TurnOffAutoMigrate(DB)
	adapter, err := gormadapter.NewAdapterByDB(DB)
	if err != nil {
		return fmt.Errorf("create casbin adapter failed: %w", err)
	}

	enforcer, err := casbin.NewSyncedEnforcer(Cfg.Casbin.ModelPath, adapter)
	if err != nil {
		return fmt.Errorf("create casbin enforcer failed: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("load casbin policy failed: %w", err)
	}

	Enforcer = enforcer
	return nil
}
