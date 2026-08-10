package ucloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	ModelDoubaoSeedream      = "doubao-seedream-4.5"
	ModelDoubaoSeedream50    = "doubao-seedream-5-0-260128"
	ModelDoubaoSeedream50Pro = "doubao-seedream-5-0-pro-260628"

	doubaoSeedreamGenerationPath = "/v1/images/generations"
)

var ErrSeedreamStreamResponse = errors.New("streaming Seedream requests must use CreateDoubaoSeedreamImageStream")

type DoubaoSeedreamGenerationRequest struct {
	Model                            string                                `json:"model"`
	Prompt                           string                                `json:"prompt"`
	Images                           []string                              `json:"images,omitempty"`
	Size                             string                                `json:"size,omitempty"`
	SequentialImageGeneration        string                                `json:"sequential_image_generation,omitempty"`
	SequentialImageGenerationOptions *DoubaoSeedreamSequentialImageOptions `json:"sequential_image_generation_options,omitempty"`
	Stream                           *bool                                 `json:"stream,omitempty"`
	ResponseFormat                   string                                `json:"response_format,omitempty"`
	Watermark                        *bool                                 `json:"watermark,omitempty"`
	OptimizePromptOptions            map[string]interface{}                `json:"optimize_prompt_options,omitempty"`
	Tools                            []DoubaoSeedreamTool                  `json:"tools,omitempty"`
	OutputFormat                     string                                `json:"output_format,omitempty"`
}

type DoubaoSeedreamParameters struct {
	Model                            string                                `json:"model"`
	Size                             string                                `json:"size,omitempty"`
	SequentialImageGeneration        string                                `json:"sequential_image_generation,omitempty"`
	SequentialImageGenerationOptions *DoubaoSeedreamSequentialImageOptions `json:"sequential_image_generation_options,omitempty"`
	Stream                           *bool                                 `json:"stream,omitempty"`
	ResponseFormat                   string                                `json:"response_format,omitempty"`
	Watermark                        *bool                                 `json:"watermark,omitempty"`
	OptimizePromptOptions            map[string]any                        `json:"optimize_prompt_options,omitempty"`
	Tools                            []DoubaoSeedreamTool                  `json:"tools,omitempty"`
	OutputFormat                     string                                `json:"output_format,omitempty"`
}

type DoubaoSeedreamSequentialImageOptions struct {
	MaxImages int `json:"max_images,omitempty"`
}

type DoubaoSeedreamTool struct {
	Type string `json:"type"`
}

type DoubaoSeedreamGenerationResponse struct {
	Model   string                `json:"model"`
	Created int64                 `json:"created"`
	Data    []DoubaoSeedreamImage `json:"data"`
	Usage   *DoubaoSeedreamUsage  `json:"usage,omitempty"`
	Error   *DoubaoSeedreamError  `json:"error,omitempty"`
}

type DoubaoSeedreamImage struct {
	URL     string               `json:"url,omitempty"`
	B64JSON string               `json:"b64_json,omitempty"`
	Size    string               `json:"size,omitempty"`
	Error   *DoubaoSeedreamError `json:"error,omitempty"`
}

type DoubaoSeedreamUsage struct {
	GeneratedImages int `json:"generated_images"`
	OutputTokens    int `json:"output_tokens"`
	TotalTokens     int `json:"total_tokens"`
}

type DoubaoSeedreamError struct {
	Code    string      `json:"code,omitempty"`
	Type    string      `json:"type,omitempty"`
	Param   interface{} `json:"param,omitempty"`
	Message string      `json:"message"`
}

func (c *Client) CreateDoubaoSeedreamImage(ctx context.Context, request DoubaoSeedreamGenerationRequest) (*DoubaoSeedreamGenerationResponse, error) {
	if request.Model == "" {
		request.Model = ModelDoubaoSeedream50
	}
	if err := validateDoubaoSeedreamRequest(&request); err != nil {
		return nil, err
	}
	if request.Stream != nil && *request.Stream {
		return nil, ErrSeedreamStreamResponse
	}
	var response DoubaoSeedreamGenerationResponse
	if err := c.postJSON(ctx, c.generationEndpoint(doubaoSeedreamGenerationPath), request, &response); err != nil {
		return nil, err
	}
	if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
		return nil, &APIError{
			StatusCode: http.StatusOK,
			Code:       response.Error.Code, Type: response.Error.Type, Param: response.Error.Param,
			Message: response.Error.Message,
		}
	}
	if response.Data == nil {
		response.Data = []DoubaoSeedreamImage{}
	}
	return &response, nil
}

// CreateDoubaoSeedreamImageStream starts an SSE image request. The caller owns
// the returned response body and must close it.
func (c *Client) CreateDoubaoSeedreamImageStream(ctx context.Context, request DoubaoSeedreamGenerationRequest) (*http.Response, error) {
	if request.Model == "" {
		request.Model = ModelDoubaoSeedream
	}
	stream := true
	request.Stream = &stream
	if err := validateDoubaoSeedreamRequest(&request); err != nil {
		return nil, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal ucloud request: %w", err)
	}
	response, err := c.post(ctx, c.generationEndpoint(doubaoSeedreamGenerationPath), bytes.NewReader(body), "text/event-stream")
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return response, nil
	}
	defer response.Body.Close()
	raw, readErr := readLimited(response.Body, c.maxResponseSize)
	if readErr != nil {
		return nil, readErr
	}
	return nil, parseAPIError(response.StatusCode, raw)
}

func buildDoubaoSeedreamRequest(request TaskSubmitRequest) (DoubaoSeedreamGenerationRequest, error) {
	var parameters DoubaoSeedreamParameters
	if err := decodeParameters(request.Parameters, &parameters); err != nil {
		return DoubaoSeedreamGenerationRequest{}, err
	}
	model := parameters.Model
	return DoubaoSeedreamGenerationRequest{
		Model: model, Prompt: request.Prompt, Images: request.Images,
		Size:                             parameters.Size,
		SequentialImageGeneration:        parameters.SequentialImageGeneration,
		SequentialImageGenerationOptions: parameters.SequentialImageGenerationOptions,
		Stream:                           parameters.Stream, ResponseFormat: parameters.ResponseFormat,
		Watermark: parameters.Watermark, OptimizePromptOptions: parameters.OptimizePromptOptions,
		Tools: parameters.Tools, OutputFormat: parameters.OutputFormat,
	}, nil
}

func validateDoubaoSeedreamRequest(request *DoubaoSeedreamGenerationRequest) error {
	request.Model = strings.TrimSpace(request.Model)
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.Size = strings.TrimSpace(request.Size)
	if !isDoubaoSeedreamModel(request.Model) {
		return fmt.Errorf("unsupported Doubao Seedream model %q", request.Model)
	}
	if request.Prompt == "" {
		return errors.New("doubao Seedream prompt is required")
	}
	maxImages := 14
	if request.Model == ModelDoubaoSeedream50Pro {
		maxImages = 10
	}
	if len(request.Images) > maxImages {
		return fmt.Errorf("doubao Seedream model %s supports at most %d reference images", request.Model, maxImages)
	}
	for i := range request.Images {
		request.Images[i] = strings.TrimSpace(request.Images[i])
		if request.Images[i] == "" {
			return fmt.Errorf("doubao Seedream images[%d] must not be empty", i)
		}
	}
	if request.Size != "" {
		if err := validateSeedreamSize(request.Size); err != nil {
			return err
		}
	}
	if request.SequentialImageGeneration != "" && !oneOf(request.SequentialImageGeneration, "auto", "disabled") {
		return errors.New("doubao Seedream sequential_image_generation must be auto or disabled")
	}
	if options := request.SequentialImageGenerationOptions; options != nil {
		if request.SequentialImageGeneration != "auto" {
			return errors.New("doubao Seedream sequential_image_generation_options requires sequential_image_generation=auto")
		}
		if options.MaxImages < 1 || options.MaxImages > 15 {
			return errors.New("doubao Seedream max_images must be between 1 and 15")
		}
	}
	if request.ResponseFormat != "" && !oneOf(request.ResponseFormat, "url", "b64_json") {
		return errors.New("doubao Seedream response_format must be url or b64_json")
	}
	if request.OutputFormat != "" && !oneOf(request.OutputFormat, "png", "jpeg") {
		return errors.New("doubao Seedream output_format must be png or jpeg")
	}
	for i := range request.Tools {
		request.Tools[i].Type = strings.TrimSpace(request.Tools[i].Type)
		if request.Tools[i].Type != "web_search" {
			return fmt.Errorf("doubao Seedream tools[%d].type must be web_search", i)
		}
	}

	switch request.Model {
	case ModelDoubaoSeedream:
		if len(request.Tools) > 0 {
			return errors.New("doubao-seedream-4.5 does not support tools")
		}
		if request.OutputFormat != "" {
			return errors.New("doubao-seedream-4.5 does not support output_format")
		}
	case ModelDoubaoSeedream50:
		// This model supports all currently documented options.
	case ModelDoubaoSeedream50Pro:
		if request.SequentialImageGeneration != "" || request.SequentialImageGenerationOptions != nil {
			return errors.New("doubao-seedream-5.0-pro does not support sequential image generation")
		}
		if request.Stream != nil {
			return errors.New("doubao-seedream-5.0-pro does not support stream")
		}
		if len(request.Tools) > 0 {
			return errors.New("doubao-seedream-5.0-pro does not support tools")
		}
		if request.OutputFormat != "" {
			return errors.New("doubao-seedream-5.0-pro does not support output_format")
		}
	}
	return nil
}

func validateSeedreamSize(size string) error {
	if strings.EqualFold(size, "2k") || strings.EqualFold(size, "4k") {
		return nil
	}
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return errors.New("doubao Seedream size must be 2K, 4K, or <width>x<height>")
	}
	width, widthErr := strconv.ParseInt(parts[0], 10, 64)
	height, heightErr := strconv.ParseInt(parts[1], 10, 64)
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return errors.New("doubao Seedream size must contain positive integer dimensions")
	}
	const minPixels int64 = 3_686_400
	const maxPixels int64 = 16_777_216
	if width > maxPixels || height > maxPixels || width > maxPixels/height {
		return errors.New("doubao Seedream size exceeds the maximum pixel count")
	}
	pixels := width * height
	if pixels < minPixels || pixels > maxPixels {
		return fmt.Errorf("doubao Seedream size must contain %d to %d pixels", minPixels, maxPixels)
	}
	if width > height*16 || height > width*16 {
		return errors.New("doubao Seedream size aspect ratio must be between 1:16 and 16:1")
	}
	return nil
}

func isDoubaoSeedreamModel(model string) bool {
	return oneOf(model, ModelDoubaoSeedream, ModelDoubaoSeedream50, ModelDoubaoSeedream50Pro)
}
