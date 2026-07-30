package ucloud

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// ModelKlingO3 is the model identifier required by the Kling O3 submit API.
	ModelKlingO3 = "kling-v3-omni"

	klingO3SubmitPath = "/v1/tasks/submit"
)

type KlingO3ImageType string

const (
	KlingO3ImageTypeFirstFrame KlingO3ImageType = "first_frame"
	KlingO3ImageTypeEndFrame   KlingO3ImageType = "end_frame"
)

type KlingO3VideoReferType string

const (
	KlingO3VideoReferTypeFeature KlingO3VideoReferType = "feature"
	KlingO3VideoReferTypeBase    KlingO3VideoReferType = "base"
)

type KlingO3SubmitRequest struct {
	Model      string            `json:"model"`
	Input      KlingO3Input      `json:"input"`
	Parameters KlingO3Parameters `json:"parameters"`
}

type KlingO3Input struct {
	Prompt         string `json:"prompt,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

type KlingO3Parameters struct {
	Mode             string                   `json:"mode,omitempty"`
	AspectRatio      string                   `json:"aspect_ratio,omitempty"`
	Duration         int                      `json:"duration,omitempty"`
	Sound            string                   `json:"sound,omitempty"`
	MultiShot        bool                     `json:"multi_shot,omitempty"`
	ShotType         string                   `json:"shot_type,omitempty"`
	MultiPrompt      []KlingO3MultiPromptItem `json:"multi_prompt,omitempty"`
	ImageList        []KlingO3ImageReference  `json:"image_list,omitempty"`
	VideoList        []KlingO3VideoReference  `json:"video_list,omitempty"`
	WatermarkEnabled *bool                    `json:"watermark_enabled,omitempty"`
	ExternalTaskID   string                   `json:"external_task_id,omitempty"`
}

type KlingO3MultiPromptItem struct {
	Index    int    `json:"index"`
	Prompt   string `json:"prompt"`
	Duration string `json:"duration"`
}

type KlingO3ImageReference struct {
	ImageURL string           `json:"image_url"`
	Type     KlingO3ImageType `json:"type,omitempty"`
}

type KlingO3VideoReference struct {
	VideoURL          string                `json:"video_url"`
	ReferType         KlingO3VideoReferType `json:"refer_type,omitempty"`
	KeepOriginalSound string                `json:"keep_original_sound,omitempty"`
}

type KlingO3SubmitResponse struct {
	Output    KlingO3SubmitOutput `json:"output"`
	RequestID string              `json:"request_id"`
}

type KlingO3SubmitOutput struct {
	TaskID string `json:"task_id"`
}

// SubmitKlingO3Task submits a Kling V3 Omni asynchronous video task.
func (c *Client) SubmitKlingO3Task(ctx context.Context, request KlingO3SubmitRequest) (*KlingO3SubmitResponse, error) {
	request.Model = ModelKlingO3
	if err := validateKlingO3Request(&request); err != nil {
		return nil, err
	}
	var response KlingO3SubmitResponse
	if err := c.postJSON(ctx, c.generationEndpoint(klingO3SubmitPath), request, &response); err != nil {
		return nil, err
	}
	response.Output.TaskID = strings.TrimSpace(response.Output.TaskID)
	if response.Output.TaskID == "" {
		return nil, errors.New("ucloud Kling O3 response does not contain task_id")
	}
	return &response, nil
}

func buildKlingO3Request(request TaskSubmitRequest) (KlingO3SubmitRequest, error) {
	var parameters KlingO3Parameters
	if err := decodeParameters(request.Parameters, &parameters); err != nil {
		return KlingO3SubmitRequest{}, err
	}
	if len(request.Images) > 0 {
		if len(parameters.ImageList) > 0 {
			return KlingO3SubmitRequest{}, errors.New("use either top-level images or parameters.image_list for Kling O3")
		}
		parameters.ImageList = make([]KlingO3ImageReference, len(request.Images))
		for i, image := range request.Images {
			parameters.ImageList[i] = KlingO3ImageReference{ImageURL: image}
		}
	}
	if request.Video != "" {
		if len(parameters.VideoList) > 0 {
			return KlingO3SubmitRequest{}, errors.New("use either top-level video or parameters.video_list for Kling O3")
		}
		parameters.VideoList = []KlingO3VideoReference{{
			VideoURL: request.Video, ReferType: KlingO3VideoReferTypeBase,
		}}
	}
	return KlingO3SubmitRequest{
		Model:      ModelKlingO3,
		Input:      KlingO3Input{Prompt: request.Prompt},
		Parameters: parameters,
	}, nil
}

func validateKlingO3Request(request *KlingO3SubmitRequest) error {
	input := &request.Input
	parameters := &request.Parameters
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.NegativePrompt = strings.TrimSpace(input.NegativePrompt)
	externalTaskID := parameters.ExternalTaskID
	parameters.ExternalTaskID = strings.TrimSpace(parameters.ExternalTaskID)

	if utf8.RuneCountInString(input.Prompt) > 2500 {
		return errors.New("kling O3 prompt must not exceed 2500 characters")
	}
	if utf8.RuneCountInString(input.NegativePrompt) > 2500 {
		return errors.New("kling O3 negative_prompt must not exceed 2500 characters")
	}
	if externalTaskID != "" && parameters.ExternalTaskID == "" {
		return errors.New("kling O3 external_task_id must not be blank")
	}
	if parameters.Mode != "" && !oneOf(parameters.Mode, "std", "pro") {
		return errors.New("kling O3 mode must be std or pro")
	}
	if parameters.AspectRatio != "" && !oneOf(parameters.AspectRatio, "16:9", "9:16", "1:1") {
		return errors.New("kling O3 aspect_ratio must be 16:9, 9:16, or 1:1")
	}
	if parameters.Sound != "" && !oneOf(parameters.Sound, "on", "off") {
		return errors.New("kling O3 sound must be on or off")
	}
	if parameters.ShotType != "" && parameters.ShotType != "customize" {
		return errors.New("kling O3 shot_type must be customize")
	}

	hasBaseVideo, err := validateKlingO3References(parameters)
	if err != nil {
		return err
	}
	if parameters.AspectRatio == "" && !hasBaseVideo {
		return errors.New("kling O3 aspect_ratio is required unless editing a base video")
	}
	if len(parameters.VideoList) > 0 && parameters.Sound == "on" {
		return errors.New("kling O3 sound must be off when video_list is present")
	}
	if parameters.Duration != 0 && !hasBaseVideo {
		maximum := 15
		if len(parameters.VideoList) > 0 {
			maximum = 10
		}
		if parameters.Duration < 3 || parameters.Duration > maximum {
			return fmt.Errorf("kling O3 duration must be between 3 and %d seconds", maximum)
		}
	}

	if parameters.MultiShot {
		if parameters.ShotType != "customize" {
			return errors.New("kling O3 multi_shot requires shot_type=customize")
		}
		if len(parameters.MultiPrompt) < 1 || len(parameters.MultiPrompt) > 6 {
			return errors.New("kling O3 multi_prompt must contain 1 to 6 shots")
		}
		if err := validateKlingO3MultiPrompt(parameters.MultiPrompt, parameters.Duration); err != nil {
			return err
		}
	} else if input.Prompt == "" {
		return errors.New("kling O3 prompt is required unless multi_shot is enabled")
	}
	return nil
}

func validateKlingO3References(parameters *KlingO3Parameters) (bool, error) {
	if len(parameters.VideoList) > 1 {
		return false, errors.New("kling O3 video_list supports at most one video")
	}
	maximumImages := 7
	if len(parameters.VideoList) > 0 {
		maximumImages = 4
	}
	if len(parameters.ImageList) > maximumImages {
		return false, fmt.Errorf("kling O3 image_list supports at most %d images for this request", maximumImages)
	}

	firstFrames := 0
	endFrames := 0
	for i := range parameters.ImageList {
		image := &parameters.ImageList[i]
		image.ImageURL = strings.TrimSpace(image.ImageURL)
		if image.ImageURL == "" {
			return false, fmt.Errorf("Kling O3 image_list[%d].image_url is required", i)
		}
		switch image.Type {
		case "":
		case KlingO3ImageTypeFirstFrame:
			firstFrames++
		case KlingO3ImageTypeEndFrame:
			endFrames++
		default:
			return false, fmt.Errorf("Kling O3 image_list[%d].type must be first_frame or end_frame", i)
		}
	}
	if firstFrames > 1 || endFrames > 1 {
		return false, errors.New("kling O3 image_list supports at most one first frame and one end frame")
	}
	if endFrames > 0 && firstFrames == 0 {
		return false, errors.New("kling O3 end_frame requires a first_frame")
	}
	if endFrames > 0 && len(parameters.ImageList) > 2 {
		return false, errors.New("kling O3 end_frame cannot be combined with more than two images")
	}

	hasBaseVideo := false
	for i := range parameters.VideoList {
		video := &parameters.VideoList[i]
		video.VideoURL = strings.TrimSpace(video.VideoURL)
		if video.VideoURL == "" || !isAbsoluteURL(video.VideoURL) {
			return false, fmt.Errorf("kling O3 video_list[%d].video_url must be an absolute URL", i)
		}
		if video.ReferType != "" && video.ReferType != KlingO3VideoReferTypeFeature && video.ReferType != KlingO3VideoReferTypeBase {
			return false, fmt.Errorf("kling O3 video_list[%d].refer_type must be feature or base", i)
		}
		if video.KeepOriginalSound != "" && !oneOf(video.KeepOriginalSound, "yes", "no") {
			return false, fmt.Errorf("kling O3 video_list[%d].keep_original_sound must be yes or no", i)
		}
		if video.ReferType == "" || video.ReferType == KlingO3VideoReferTypeBase {
			hasBaseVideo = true
		}
	}
	return hasBaseVideo, nil
}

func validateKlingO3MultiPrompt(prompts []KlingO3MultiPromptItem, duration int) error {
	expectedDuration := duration
	if expectedDuration == 0 {
		expectedDuration = 5
	}
	totalDuration := 0
	seenIndexes := make(map[int]struct{}, len(prompts))
	for i := range prompts {
		prompt := &prompts[i]
		prompt.Prompt = strings.TrimSpace(prompt.Prompt)
		prompt.Duration = strings.TrimSpace(prompt.Duration)
		if prompt.Index < 1 {
			return fmt.Errorf("kling O3 multi_prompt[%d].index must be at least 1", i)
		}
		if _, exists := seenIndexes[prompt.Index]; exists {
			return fmt.Errorf("kling O3 multi_prompt index %d is duplicated", prompt.Index)
		}
		seenIndexes[prompt.Index] = struct{}{}
		if prompt.Prompt == "" || utf8.RuneCountInString(prompt.Prompt) > 512 {
			return fmt.Errorf("kling O3 multi_prompt[%d].prompt must contain 1 to 512 characters", i)
		}
		seconds, err := strconv.Atoi(prompt.Duration)
		if err != nil || seconds < 1 || seconds > expectedDuration {
			return fmt.Errorf("kling O3 multi_prompt[%d].duration must be an integer from 1 to %d", i, expectedDuration)
		}
		totalDuration += seconds
	}
	if totalDuration != expectedDuration {
		return fmt.Errorf("kling O3 multi_prompt durations total %d, want %d", totalDuration, expectedDuration)
	}
	return nil
}
