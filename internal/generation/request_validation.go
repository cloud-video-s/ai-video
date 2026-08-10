package generation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"ai-video/internal/gen/model"
)

func generationInputFromMap(taskType uint32, source map[string]any) (GenerationInput, error) {
	raw, err := json.Marshal(source)
	if err != nil {
		return GenerationInput{}, fmt.Errorf("encode input: %w", err)
	}
	var input GenerationInput
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return GenerationInput{}, fmt.Errorf("invalid input: %w", err)
	}
	if err := normalizeGenerationInput(taskType, &input); err != nil {
		return GenerationInput{}, err
	}
	return input, nil
}

func normalizeGenerationInput(taskType uint32, input *GenerationInput) error {
	if input == nil {
		return errors.New("input is required")
	}
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.Video = strings.TrimSpace(input.Video)
	input.FirstFrame = strings.TrimSpace(input.FirstFrame)
	input.EndFrame = strings.TrimSpace(input.EndFrame)
	if input.Prompt == "" {
		return errors.New("input.prompt is required")
	}
	for i := range input.Images {
		input.Images[i] = strings.TrimSpace(input.Images[i])
		if input.Images[i] == "" {
			return fmt.Errorf("input.images[%d] must not be empty", i)
		}
	}
	switch taskType {
	case TaskTypeImage:
		if input.Video != "" || input.FirstFrame != "" || input.EndFrame != "" {
			return errors.New("image generation only supports prompt and reference images")
		}
	case TaskTypeVideo:
		if input.EndFrame != "" && input.FirstFrame == "" {
			return errors.New("input.end_frame requires input.first_frame")
		}
		if input.FirstFrame != "" && (len(input.Images) > 0 || input.Video != "") {
			return errors.New("first/end frame generation cannot be combined with input.images or input.video")
		}
	default:
		return fmt.Errorf("unsupported task_type %d", taskType)
	}
	return nil
}

func mergeConfiguredParameters(definitions []model.VideoModelParameter, request map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	configured := make(map[string]model.VideoModelParameter)
	for i := range definitions {
		definition := definitions[i]
		if definition.ParameterType == 2 {
			continue
		}
		key := strings.TrimSpace(definition.ParamKey)
		if key == "" {
			return nil, errors.New("model parameter contains an empty param_key")
		}
		if _, exists := configured[key]; exists {
			return nil, fmt.Errorf("model parameter %s is configured more than once", key)
		}
		configured[key] = definition
		raw := strings.TrimSpace(definition.DefaultValue)
		if raw == "" {
			continue
		}
		var value any
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return nil, fmt.Errorf("model parameter %s has invalid default JSON: %w", key, err)
		}
		if err := validateConfiguredParameter(definition, value); err != nil {
			return nil, fmt.Errorf("model parameter %s default is invalid: %w", key, err)
		}
		result[key] = value
	}
	for rawKey, value := range request {
		key := strings.TrimSpace(rawKey)
		definition, exists := configured[key]
		if !exists || key != rawKey {
			return nil, fmt.Errorf("parameter %q is not configured for the selected model", rawKey)
		}
		if err := validateConfiguredParameter(definition, value); err != nil {
			return nil, fmt.Errorf("parameter %s is invalid: %w", key, err)
		}
		result[key] = value
	}
	for key, definition := range configured {
		if definition.IsRequired == 1 {
			if _, exists := result[key]; !exists {
				return nil, fmt.Errorf("parameter %s is required", key)
			}
		}
	}
	if stream, ok := result["stream"].(bool); ok && stream {
		return nil, errors.New("parameter stream=true is not supported by the task API")
	}
	return result, nil
}

func validateConfiguredParameter(definition model.VideoModelParameter, value interface{}) error {
	switch strings.ToLower(strings.TrimSpace(definition.ParamType)) {
	case "", "any":
	case "string":
		if _, ok := value.(string); !ok {
			return errors.New("must be a string")
		}
	case "integer":
		number, ok := jsonNumber(value)
		if !ok || math.Trunc(number) != number {
			return errors.New("must be an integer")
		}
	case "number":
		if _, ok := jsonNumber(value); !ok {
			return errors.New("must be a number")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("must be a boolean")
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return errors.New("must be an object")
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return errors.New("must be an array")
		}
	default:
		return fmt.Errorf("uses unsupported value type %q", definition.ParamType)
	}
	if raw := strings.TrimSpace(definition.AllowedValues); raw != "" && raw != "[]" {
		var allowed []any
		if err := json.Unmarshal([]byte(raw), &allowed); err != nil {
			return fmt.Errorf("allowed_values is invalid JSON: %w", err)
		}
		matched := false
		candidate, _ := json.Marshal(value)
		for _, allowedValue := range allowed {
			encoded, _ := json.Marshal(allowedValue)
			if string(encoded) == string(candidate) {
				matched = true
				break
			}
		}
		if !matched {
			return errors.New("is not in allowed_values")
		}
	}
	return validateConfiguredConstraints(definition.Constraints, value)
}

func validateConfiguredConstraints(raw string, value any) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var constraints map[string]any
	if err := json.Unmarshal([]byte(raw), &constraints); err != nil {
		return fmt.Errorf("constraints is invalid JSON: %w", err)
	}
	if number, ok := jsonNumber(value); ok {
		if minimum, exists := jsonNumber(constraints["min"]); exists && number < minimum {
			return fmt.Errorf("must be at least %g", minimum)
		}
		if maximum, exists := jsonNumber(constraints["max"]); exists && number > maximum {
			return fmt.Errorf("must be at most %g", maximum)
		}
	}
	length := -1
	if text, ok := value.(string); ok {
		length = len([]rune(text))
	} else if values, ok := value.([]any); ok {
		length = len(values)
	}
	if length >= 0 {
		if minimum, exists := jsonNumber(constraints["min_length"]); exists && float64(length) < minimum {
			return fmt.Errorf("length must be at least %g", minimum)
		}
		if maximum, exists := jsonNumber(constraints["max_length"]); exists && float64(length) > maximum {
			return fmt.Errorf("length must be at most %g", maximum)
		}
	}
	if pattern, ok := constraints["pattern"].(string); ok {
		text, ok := value.(string)
		if !ok {
			return errors.New("pattern constraint requires a string value")
		}
		matched, err := regexp.MatchString(pattern, text)
		if err != nil {
			return fmt.Errorf("pattern constraint is invalid: %w", err)
		}
		if !matched {
			return errors.New("does not match the configured pattern")
		}
	}
	return nil
}

func jsonNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int8:
		return float64(number), true
	case int16:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint8:
		return float64(number), true
	case uint16:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}
