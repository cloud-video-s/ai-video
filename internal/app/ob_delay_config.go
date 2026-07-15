package app

import (
	"fmt"
	"os"
	"strings"

	"ai-video/internal/model"

	"github.com/spf13/viper"
	"gorm.io/gorm/clause"
)

const DefaultOBDelayConfigPath = "config/ob_delay_config.yaml"

type obDelayConfigFile struct {
	Configs []obDelayConfigItem `mapstructure:"configs"`
}

type obDelayConfigItem struct {
	Group   string `mapstructure:"group"`
	Key     string `mapstructure:"key"`
	Value   string `mapstructure:"value"`
	Type    string `mapstructure:"type"`
	Options string `mapstructure:"options"`
	Remark  string `mapstructure:"remark"`
	Sort    int    `mapstructure:"sort"`
}

func SeedOBDelayConfig(path string) error {
	items, err := loadOBDelayConfig(path)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	rows := make([]model.VideoDelayConfig, 0, len(items))
	for i := range items {
		it := items[i]
		if err := validateOBDelayConfigItem(it); err != nil {
			return err
		}
		rows = append(rows, model.VideoDelayConfig{
			Group:   strings.TrimSpace(it.Group),
			Key:     strings.TrimSpace(it.Key),
			Value:   strings.TrimSpace(it.Value),
			Type:    strings.TrimSpace(it.Type),
			Options: strings.TrimSpace(it.Options),
			Remark:  strings.TrimSpace(it.Remark),
			Sort:    it.Sort,
		})
	}

	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"group", "value", "type", "options", "remark", "sort",
		}),
	}).Create(&rows).Error
}

func loadOBDelayConfig(path string) ([]obDelayConfigItem, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultOBDelayConfigPath
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("read ob delay config %q: %w", path, err)
	}
	var cfg obDelayConfigFile
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read ob delay config %q: %w", path, err)
	}
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parse ob delay config %q: %w", path, err)
	}
	return cfg.Configs, nil
}

func validateOBDelayConfigItem(it obDelayConfigItem) error {
	if strings.TrimSpace(it.Group) == "" {
		return fmt.Errorf("ob delay config %q group is required", it.Key)
	}
	if strings.TrimSpace(it.Key) == "" {
		return fmt.Errorf("ob delay config key is required")
	}
	if strings.TrimSpace(it.Value) == "" {
		return fmt.Errorf("ob delay config %q value is required", it.Key)
	}
	switch strings.TrimSpace(it.Type) {
	case "string", "int", "bool":
		return nil
	default:
		return fmt.Errorf("ob delay config %q type is unsupported: %s", it.Key, it.Type)
	}
}
