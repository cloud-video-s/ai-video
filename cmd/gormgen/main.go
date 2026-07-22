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
	models, err := generateModelsWithRelations(g, db)
	if err != nil {
		panic(fmt.Sprintf("inspect database relationships failed: %v", err))
	}
	g.ApplyBasic(models...)

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
