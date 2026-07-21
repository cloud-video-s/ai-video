package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestTemplateDisplayConfigReferencesPositionKey(t *testing.T) {
	parsed, err := schema.Parse(&VideoTemplateDisplayConfig{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	relation := parsed.Relationships.Relations["DisplayPosition"]
	if relation == nil || len(relation.References) != 1 {
		t.Fatalf("display-position relationship = %#v", relation)
	}
	if relation.Type != schema.BelongsTo {
		t.Fatalf("display-position relationship type = %s, want belongs_to", relation.Type)
	}
	reference := relation.References[0]
	if reference.ForeignKey == nil || reference.ForeignKey.Name != "DisplayPositionKey" {
		t.Fatalf("foreign key = %#v, want DisplayPositionKey", reference.ForeignKey)
	}
	if reference.PrimaryKey == nil || reference.PrimaryKey.Name != "PositionKey" {
		t.Fatalf("referenced key = %#v, want PositionKey", reference.PrimaryKey)
	}
	if reference.ForeignKey.DataType != schema.String || reference.PrimaryKey.DataType != schema.String {
		t.Fatalf("position-key relationship types = %s -> %s, want string -> string", reference.ForeignKey.DataType, reference.PrimaryKey.DataType)
	}
	constraint := relation.ParseConstraint()
	if constraint == nil {
		t.Fatal("display-position foreign-key constraint is missing")
	}
	if constraint.Schema == nil || constraint.Schema.Table != "video_template_display_config" {
		t.Fatalf("foreign-key owner table = %#v, want video_template_display_config", constraint.Schema)
	}
	if constraint.ReferenceSchema == nil || constraint.ReferenceSchema.Table != "video_display_position" {
		t.Fatalf("foreign-key referenced table = %#v, want video_display_position", constraint.ReferenceSchema)
	}
	if len(constraint.ForeignKeys) != 1 || constraint.ForeignKeys[0].DBName != "position_key" ||
		len(constraint.References) != 1 || constraint.References[0].DBName != "position_key" {
		t.Fatalf("foreign-key columns = %#v -> %#v, want position_key -> position_key", constraint.ForeignKeys, constraint.References)
	}
}
