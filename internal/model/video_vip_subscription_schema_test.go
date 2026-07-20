package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestVideoVIPSubscriptionSchemaAssociations(t *testing.T) {
	parsed, err := schema.Parse(&VideoVIPSubscription{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Table != "video_vip_subscription" {
		t.Fatalf("table = %q", parsed.Table)
	}
	wantJoinTables := map[string]string{
		"DisplayPositions": "video_vip_subscription_position",
		"Channels":         "video_vip_subscription_channel",
		"ExcludedChannels": "video_vip_subscription_excluded_channel",
	}
	for association, wantTable := range wantJoinTables {
		relation := parsed.Relationships.Relations[association]
		if relation == nil || relation.JoinTable == nil {
			t.Fatalf("association %s has no join table", association)
		}
		if relation.JoinTable.Table != wantTable {
			t.Fatalf("association %s join table = %q, want %q", association, relation.JoinTable.Table, wantTable)
		}
	}

	assertJoinColumns(t, parsed.Relationships.Relations["DisplayPositions"].JoinTable,
		"subscription_id", "display_position_id")
	assertJoinColumns(t, parsed.Relationships.Relations["Channels"].JoinTable,
		"subscription_id", "channel_id")
	assertJoinColumns(t, parsed.Relationships.Relations["ExcludedChannels"].JoinTable,
		"subscription_id", "channel_id")
}

func assertJoinColumns(t *testing.T, join *schema.Schema, columns ...string) {
	t.Helper()
	for _, column := range columns {
		if join.LookUpField(column) == nil {
			available := make([]string, 0, len(join.Fields))
			for _, field := range join.Fields {
				available = append(available, field.DBName)
			}
			t.Fatalf("join table %s does not contain column %s; available: %v", join.Table, column, available)
		}
	}
}
