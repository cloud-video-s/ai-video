package repository

import (
	"encoding/json"
	"testing"
)

func TestDelayConfigValueJSONContainsOnlyKeyAndValue(t *testing.T) {
	data, err := json.Marshal(DelayConfigValue{Key: "OBPaymentCloseDely", Value: "5"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"key":"OBPaymentCloseDely","value":"5"}` {
		t.Fatalf("unexpected JSON: %s", data)
	}
}
