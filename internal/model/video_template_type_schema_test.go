package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestVideoTemplateTypeUsesPositionKeyAssociation(t *testing.T) {
	parsed, err := schema.Parse(&VideoTemplateType{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, fieldName := range []string{"LegacyCountry", "LegacyAppPackage", "LegacyChannelID", "LegacyPackageID", "UserTypes", "SubscriptionStatuses"} {
		if parsed.FieldsByName[fieldName] == nil {
			t.Fatalf("field %s is missing", fieldName)
		}
	}
	relation := parsed.Relationships.Relations["DisplayPositions"]
	if relation == nil || relation.JoinTable == nil {
		t.Fatal("display positions association is missing")
	}
	if relation.JoinTable.Table != "video_template_type_display_position" {
		t.Fatalf("join table=%q", relation.JoinTable.Table)
	}
	for _, column := range []string{"template_type_id", "position_key"} {
		if relation.JoinTable.LookUpField(column) == nil {
			t.Fatalf("join column %s is missing", column)
		}
	}
	for association, table := range map[string]string{
		"Countries": "video_template_type_country",
		"Channels":  "video_template_type_channel",
		"Packages":  "video_template_type_package",
	} {
		relation := parsed.Relationships.Relations[association]
		if relation == nil || relation.JoinTable == nil || relation.JoinTable.Table != table {
			t.Fatalf("association %s join table = %#v", association, relation)
		}
	}
}
