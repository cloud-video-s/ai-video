package modelparameter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type ValueOption struct {
	Value interface{} `json:"value"`
	Alias string      `json:"alias"`
}

// ValueOptions parses the canonical [{"value":...,"alias":"..."}] format.
// Legacy string arrays are combined with allowedValues for backward
// compatibility and become explicit pairs the next time the row is saved.
func ValueOptions(raw string, allowedValues []interface{}) ([]ValueOption, error) {
	if strings.TrimSpace(raw) == "" {
		options := make([]ValueOption, len(allowedValues))
		for i := range allowedValues {
			options[i] = ValueOption{Value: allowedValues[i], Alias: displayValue(allowedValues[i])}
		}
		return options, nil
	}
	var options []ValueOption
	if err := json.Unmarshal([]byte(raw), &options); err == nil {
		if len(options) != len(allowedValues) {
			return nil, errors.New("选择值配置与允许值数量不一致")
		}
		for i := range options {
			if !valuesEqual(options[i].Value, allowedValues[i]) {
				return nil, fmt.Errorf("第 %d 个选择值配置与允许值不一致", i+1)
			}
		}
		return options, nil
	}

	var aliases []string
	if err := json.Unmarshal([]byte(raw), &aliases); err != nil {
		return nil, fmt.Errorf("选择值配置必须是 value/alias 对象数组: %w", err)
	}
	if len(aliases) != len(allowedValues) {
		return nil, errors.New("选择值与别名数量不一致")
	}
	options = make([]ValueOption, len(allowedValues))
	for i := range allowedValues {
		options[i] = ValueOption{Value: allowedValues[i], Alias: aliases[i]}
	}
	return options, nil
}

func Values(options []ValueOption) []interface{} {
	values := make([]interface{}, len(options))
	for i := range options {
		values[i] = options[i].Value
	}
	return values
}

func valuesEqual(left, right interface{}) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func displayValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}
