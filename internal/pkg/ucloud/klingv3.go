package ucloud

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	ModelKlingV3 = "kling-v3"

	klingV3SubmitPath = "/v1/tasks/submit"
)

type KlingV3Type string

const (
	KlingV3TypeTextToVideo   KlingV3Type = "t2v"
	KlingV3TypeImageToVideo  KlingV3Type = "i2v"
	KlingV3TypeMotionControl KlingV3Type = "motion_control"
)

type KlingV3TaskSubmitRequest struct {
	Model      string            `json:"model"`
	Input      KlingV3Input      `json:"input,omitempty"`
	Parameters KlingV3Parameters `json:"parameters,omitempty"`
}

type KlingV3Input struct {
	Prompt         string   `json:"prompt,omitempty"`
	NegativePrompt string   `json:"negative_prompt,omitempty"`
	ImgURL         string   `json:"img_url,omitempty"`
	VideoURL       string   `json:"video_url,omitempty"`
	FirstFrameURL  string   `json:"first_frame_url,omitempty"`
	Images         []string `json:"images,omitempty"`
}

type KlingV3Parameters struct {
	KlingV3Type          KlingV3Type          `json:"kling_v3_type,omitempty"`
	Mode                 string               `json:"mode,omitempty"`
	AspectRatio          string               `json:"aspect_ratio,omitempty"`
	Duration             int                  `json:"duration,omitempty"`
	WatermarkEnabled     *bool                `json:"watermark_enabled,omitempty"`
	ExternalTaskID       string               `json:"external_task_id,omitempty"`
	Image                string               `json:"image,omitempty"`
	ImageTail            string               `json:"image_tail,omitempty"`
	Sound                string               `json:"sound,omitempty"`
	MultiShot            bool                 `json:"multi_shot,omitempty"`
	ShotType             string               `json:"shot_type,omitempty"`
	MultiPrompt          []KlingV3MultiPrompt `json:"multi_prompt,omitempty"`
	CharacterOrientation string               `json:"character_orientation,omitempty"`
	KeepOriginalSound    string               `json:"keep_original_sound,omitempty"`
}

type KlingV3MultiPrompt struct {
	Index    int    `json:"index"`
	Prompt   string `json:"prompt"`
	Duration string `json:"duration"`
}

type KlingV3TaskSubmitResponse struct {
	Output    KlingV3TaskSubmitOutput `json:"output"`
	RequestID string                  `json:"request_id"`
}

type KlingV3TaskSubmitOutput struct {
	TaskID string `json:"task_id"`
}

func (c *Client) SubmitKlingV3Task(ctx context.Context, request KlingV3TaskSubmitRequest) (*KlingV3TaskSubmitResponse, error) {
	request.Model = ModelKlingV3
	if err := validateKlingV3Request(&request); err != nil {
		return nil, err
	}
	var response KlingV3TaskSubmitResponse
	if err := c.postJSON(ctx, klingV3SubmitPath, request, &response); err != nil {
		return nil, err
	}
	response.Output.TaskID = strings.TrimSpace(response.Output.TaskID)
	if response.Output.TaskID == "" {
		return nil, errors.New("ucloud Kling v3 response does not contain task_id")
	}
	return &response, nil
}

func buildKlingV3Request(request TaskSubmitRequest) (KlingV3TaskSubmitRequest, error) {
	var parameters KlingV3Parameters
	if err := decodeParameters(request.Parameters, &parameters); err != nil {
		return KlingV3TaskSubmitRequest{}, err
	}
	if len(request.Images) > 2 {
		return KlingV3TaskSubmitRequest{}, errors.New("Kling v3 supports at most two images (first and tail frame)")
	}
	input := KlingV3Input{Prompt: request.Prompt}

	if request.Video != "" {
		if len(request.Images) != 1 {
			return KlingV3TaskSubmitRequest{}, errors.New("Kling v3 motion control requires exactly one reference image")
		}
		if parameters.Image != "" || parameters.ImageTail != "" {
			return KlingV3TaskSubmitRequest{}, errors.New("use the top-level images field for motion-control reference images")
		}
		if parameters.KlingV3Type != "" && parameters.KlingV3Type != KlingV3TypeMotionControl {
			return KlingV3TaskSubmitRequest{}, errors.New("video input requires kling_v3_type=motion_control")
		}
		parameters.KlingV3Type = KlingV3TypeMotionControl
		input.ImgURL = request.Images[0]
		input.VideoURL = request.Video
	} else if len(request.Images) > 0 {
		if parameters.KlingV3Type != "" && parameters.KlingV3Type != KlingV3TypeImageToVideo {
			return KlingV3TaskSubmitRequest{}, errors.New("image input requires kling_v3_type=i2v")
		}
		parameters.KlingV3Type = KlingV3TypeImageToVideo
		if parameters.Image != "" && parameters.Image != request.Images[0] {
			return KlingV3TaskSubmitRequest{}, errors.New("parameters.image conflicts with images[0]")
		}
		parameters.Image = request.Images[0]
		if len(request.Images) == 2 {
			if parameters.ImageTail != "" && parameters.ImageTail != request.Images[1] {
				return KlingV3TaskSubmitRequest{}, errors.New("parameters.image_tail conflicts with images[1]")
			}
			parameters.ImageTail = request.Images[1]
		}
	} else if parameters.Image != "" {
		if parameters.KlingV3Type != "" && parameters.KlingV3Type != KlingV3TypeImageToVideo {
			return KlingV3TaskSubmitRequest{}, errors.New("parameters.image requires kling_v3_type=i2v")
		}
		parameters.KlingV3Type = KlingV3TypeImageToVideo
	} else if parameters.KlingV3Type == "" {
		parameters.KlingV3Type = KlingV3TypeTextToVideo
	}

	return KlingV3TaskSubmitRequest{
		Model: ModelKlingV3, Input: input, Parameters: parameters,
	}, nil
}

func validateKlingV3Request(request *KlingV3TaskSubmitRequest) error {
	input := &request.Input
	parameters := &request.Parameters
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.NegativePrompt = strings.TrimSpace(input.NegativePrompt)
	input.ImgURL = strings.TrimSpace(input.ImgURL)
	input.VideoURL = strings.TrimSpace(input.VideoURL)
	input.FirstFrameURL = strings.TrimSpace(input.FirstFrameURL)
	parameters.Image = strings.TrimSpace(parameters.Image)
	parameters.ImageTail = strings.TrimSpace(parameters.ImageTail)
	parameters.ExternalTaskID = strings.TrimSpace(parameters.ExternalTaskID)

	if utf8.RuneCountInString(input.Prompt) > 2500 {
		return errors.New("Kling v3 prompt must not exceed 2500 characters")
	}
	if utf8.RuneCountInString(input.NegativePrompt) > 2500 {
		return errors.New("Kling v3 negative_prompt must not exceed 2500 characters")
	}
	for i := range input.Images {
		input.Images[i] = strings.TrimSpace(input.Images[i])
		if input.Images[i] == "" {
			return fmt.Errorf("Kling v3 input.images[%d] must not be empty", i)
		}
	}
	if input.VideoURL != "" {
		if !isAbsoluteURL(input.VideoURL) {
			return errors.New("Kling v3 video_url must be an absolute URL")
		}
	}
	if input.FirstFrameURL != "" && !isAbsoluteURL(input.FirstFrameURL) {
		return errors.New("Kling v3 first_frame_url must be an absolute URL")
	}

	generationMode := parameters.KlingV3Type
	if generationMode == "" {
		switch {
		case input.VideoURL != "":
			generationMode = KlingV3TypeMotionControl
		case hasKlingV3Image(*input, *parameters):
			generationMode = KlingV3TypeImageToVideo
		default:
			generationMode = KlingV3TypeTextToVideo
		}
		parameters.KlingV3Type = generationMode
	}
	if !oneOf(string(generationMode), string(KlingV3TypeTextToVideo), string(KlingV3TypeImageToVideo), string(KlingV3TypeMotionControl)) {
		return fmt.Errorf("unsupported Kling v3 type %q", generationMode)
	}
	if parameters.Mode != "" && !oneOf(parameters.Mode, "std", "pro") {
		return errors.New("Kling v3 mode must be std or pro")
	}
	if parameters.AspectRatio != "" && !oneOf(parameters.AspectRatio, "16:9", "9:16", "1:1") {
		return errors.New("Kling v3 aspect_ratio must be 16:9, 9:16, or 1:1")
	}
	if parameters.Sound != "" && !oneOf(parameters.Sound, "on", "off") {
		return errors.New("Kling v3 sound must be on or off")
	}
	if parameters.KeepOriginalSound != "" && !oneOf(parameters.KeepOriginalSound, "yes", "no") {
		return errors.New("Kling v3 keep_original_sound must be yes or no")
	}
	if parameters.ImageTail != "" && parameters.Image == "" {
		return errors.New("Kling v3 image_tail requires image")
	}

	switch generationMode {
	case KlingV3TypeTextToVideo:
		if hasKlingV3Image(*input, *parameters) || input.VideoURL != "" {
			return errors.New("Kling v3 t2v does not accept image or video input")
		}
	case KlingV3TypeImageToVideo:
		if !hasKlingV3Image(*input, *parameters) {
			return errors.New("Kling v3 i2v requires an image")
		}
		if input.VideoURL != "" {
			return errors.New("Kling v3 i2v does not accept video input")
		}
	case KlingV3TypeMotionControl:
		if input.ImgURL == "" || input.VideoURL == "" {
			return errors.New("Kling v3 motion_control requires img_url and video_url")
		}
		if !oneOf(parameters.CharacterOrientation, "image", "video") {
			return errors.New("Kling v3 motion_control requires character_orientation=image or video")
		}
	}

	if generationMode != KlingV3TypeMotionControl && parameters.CharacterOrientation != "" {
		return errors.New("Kling v3 character_orientation is only valid for motion_control")
	}
	if generationMode == KlingV3TypeMotionControl {
		if parameters.Duration != 0 && parameters.Duration != 5 && parameters.Duration != 10 {
			return errors.New("Kling v3 motion_control duration must be 5 or 10 seconds")
		}
	} else if parameters.Duration != 0 && (parameters.Duration < 3 || parameters.Duration > 15) {
		return errors.New("Kling v3 duration must be between 3 and 15 seconds")
	}

	if parameters.MultiShot {
		if generationMode != KlingV3TypeTextToVideo {
			return errors.New("Kling v3 multi_shot is only supported for t2v")
		}
		if parameters.ShotType != "customize" {
			return errors.New("Kling v3 multi_shot requires shot_type=customize")
		}
		if len(parameters.MultiPrompt) < 1 || len(parameters.MultiPrompt) > 6 {
			return errors.New("Kling v3 multi_prompt must contain 1 to 6 shots")
		}
		if err := validateKlingV3MultiPrompt(parameters.MultiPrompt, parameters.Duration); err != nil {
			return err
		}
	} else if input.Prompt == "" {
		return errors.New("Kling v3 prompt is required unless multi_shot is enabled")
	}
	return nil
}

func validateKlingV3MultiPrompt(prompts []KlingV3MultiPrompt, duration int) error {
	total := 0
	seen := make(map[int]struct{}, len(prompts))
	for i := range prompts {
		prompt := &prompts[i]
		prompt.Prompt = strings.TrimSpace(prompt.Prompt)
		prompt.Duration = strings.TrimSpace(prompt.Duration)
		if prompt.Index < 1 {
			return fmt.Errorf("Kling v3 multi_prompt[%d].index must be at least 1", i)
		}
		if _, exists := seen[prompt.Index]; exists {
			return fmt.Errorf("Kling v3 multi_prompt index %d is duplicated", prompt.Index)
		}
		seen[prompt.Index] = struct{}{}
		if prompt.Prompt == "" || utf8.RuneCountInString(prompt.Prompt) > 512 {
			return fmt.Errorf("Kling v3 multi_prompt[%d].prompt must contain 1 to 512 characters", i)
		}
		seconds, err := strconv.Atoi(prompt.Duration)
		if err != nil || seconds < 1 {
			return fmt.Errorf("Kling v3 multi_prompt[%d].duration must be a positive integer string", i)
		}
		total += seconds
	}
	expected := duration
	if expected == 0 {
		expected = 5
	}
	if total != expected {
		return fmt.Errorf("Kling v3 multi_prompt durations total %d, want %d", total, expected)
	}
	return nil
}

func hasKlingV3Image(input KlingV3Input, parameters KlingV3Parameters) bool {
	return input.ImgURL != "" || input.FirstFrameURL != "" || len(input.Images) > 0 || parameters.Image != ""
}

func isAbsoluteURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
