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
	// ModelKlingV3 is the model identifier required by the Kling V3 API.
	ModelKlingV3 = "kling-v3"

	klingV3SubmitPath = "/v1/tasks/submit"
	klingV3StatusPath = "/v1/tasks/status"
)

type KlingV3GenerationType string

const (
	KlingV3TypeT2V           KlingV3GenerationType = "t2v"
	KlingV3TypeI2V           KlingV3GenerationType = "i2v"
	KlingV3TypeMotionControl KlingV3GenerationType = "motion_control"
)

type KlingV3TaskStatus string

const (
	KlingV3TaskStatusPending KlingV3TaskStatus = "Pending"
	KlingV3TaskStatusRunning KlingV3TaskStatus = "Running"
	KlingV3TaskStatusSuccess KlingV3TaskStatus = "Success"
	KlingV3TaskStatusFailure KlingV3TaskStatus = "Failure"
)

type KlingV3SubmitRequest struct {
	Model      string            `json:"model"`
	Input      KlingV3Input      `json:"input"`
	Parameters KlingV3Parameters `json:"parameters"`
}

type KlingV3Input struct {
	FirstFrameURL  string   `json:"first_frame_url,omitempty"`
	Images         []string `json:"images,omitempty"`
	ImgURL         string   `json:"img_url,omitempty"`
	NegativePrompt string   `json:"negative_prompt,omitempty"`
	Prompt         string   `json:"prompt,omitempty"`
	VideoURL       string   `json:"video_url,omitempty"`
}

type KlingV3Parameters struct {
	AspectRatio          string                   `json:"aspect_ratio,omitempty"`
	CharacterOrientation string                   `json:"character_orientation,omitempty"`
	Duration             int                      `json:"duration,omitempty"`
	ExternalTaskID       string                   `json:"external_task_id,omitempty"`
	Image                string                   `json:"image,omitempty"`
	ImageTail            string                   `json:"image_tail,omitempty"`
	KeepOriginalSound    string                   `json:"keep_original_sound,omitempty"`
	KlingV3Type          KlingV3GenerationType    `json:"kling_v3_type,omitempty"`
	Mode                 string                   `json:"mode,omitempty"`
	MultiPrompt          []KlingV3MultiPromptItem `json:"multi_prompt,omitempty"`
	MultiShot            bool                     `json:"multi_shot,omitempty"`
	ShotType             string                   `json:"shot_type,omitempty"`
	Sound                string                   `json:"sound,omitempty"`
	WatermarkEnabled     *bool                    `json:"watermark_enabled,omitempty"`
}

type KlingV3MultiPromptItem struct {
	Duration string `json:"duration"`
	Index    int    `json:"index"`
	Prompt   string `json:"prompt"`
}

type KlingV3SubmitResponse struct {
	Output    KlingV3SubmitOutput `json:"output"`
	RequestID string              `json:"request_id"`
}

type KlingV3SubmitOutput struct {
	TaskID string `json:"task_id"`
}

type KlingV3StatusResponse struct {
	Output    KlingV3StatusOutput `json:"output"`
	Usage     *KlingV3Usage       `json:"usage,omitempty"`
	RequestID string              `json:"request_id"`
}

type KlingV3StatusOutput struct {
	TaskID       string            `json:"task_id"`
	TaskStatus   KlingV3TaskStatus `json:"task_status"`
	URLs         []string          `json:"urls,omitempty"`
	SubmitTime   int64             `json:"submit_time,omitempty"`
	FinishTime   int64             `json:"finish_time,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type KlingV3Usage struct {
	Duration int `json:"duration"`
}

// SubmitKlingV3Task submits a Kling V3 asynchronous video task.
func (c *Client) SubmitKlingV3Task(ctx context.Context, request KlingV3SubmitRequest) (*KlingV3SubmitResponse, error) {
	request.Model = ModelKlingV3
	if err := validateKlingV3Request(&request); err != nil {
		return nil, err
	}
	var response KlingV3SubmitResponse
	if err := c.postJSON(ctx, c.generationEndpoint(klingV3SubmitPath), request, &response); err != nil {
		return nil, err
	}
	response.Output.TaskID = strings.TrimSpace(response.Output.TaskID)
	if response.Output.TaskID == "" {
		return nil, errors.New("ucloud Kling V3 response does not contain task_id")
	}
	return &response, nil
}

// GetKlingV3TaskStatus returns the current state and output of a Kling V3 task.
func (c *Client) GetKlingV3TaskStatus(ctx context.Context, taskID string) (*KlingV3StatusResponse, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errors.New("kling V3 task_id is required")
	}
	query := make(url.Values, 1)
	query.Set("task_id", taskID)
	var response KlingV3StatusResponse
	if err := c.getJSON(ctx, c.taskStatusEndpoint(klingV3StatusPath), query, &response); err != nil {
		return nil, err
	}
	response.Output.TaskID = strings.TrimSpace(response.Output.TaskID)
	response.Output.TaskStatus = KlingV3TaskStatus(strings.TrimSpace(string(response.Output.TaskStatus)))
	response.Output.ErrorMessage = strings.TrimSpace(response.Output.ErrorMessage)
	if response.Output.TaskID == "" {
		return nil, errors.New("ucloud Kling V3 status response does not contain task_id")
	}
	if response.Output.TaskID != taskID {
		return nil, fmt.Errorf("ucloud Kling V3 status task_id %q does not match requested task_id %q", response.Output.TaskID, taskID)
	}
	if !oneOf(string(response.Output.TaskStatus),
		string(KlingV3TaskStatusPending), string(KlingV3TaskStatusRunning),
		string(KlingV3TaskStatusSuccess), string(KlingV3TaskStatusFailure)) {
		return nil, fmt.Errorf("ucloud Kling V3 returned unsupported task_status %q", response.Output.TaskStatus)
	}
	for i := range response.Output.URLs {
		response.Output.URLs[i] = strings.TrimSpace(response.Output.URLs[i])
		if response.Output.URLs[i] == "" {
			return nil, fmt.Errorf("ucloud Kling V3 status urls[%d] must not be empty", i)
		}
	}
	return &response, nil
}

func buildKlingV3Request(request TaskSubmitRequest) (KlingV3SubmitRequest, error) {
	var parameters KlingV3Parameters
	if err := decodeParameters(request.Parameters, &parameters); err != nil {
		return KlingV3SubmitRequest{}, err
	}
	imgURL := ""
	if len(request.Images) > 0 {
		imgURL = request.Images[0]
	}
	return KlingV3SubmitRequest{
		Model: ModelKlingV3,
		Input: KlingV3Input{
			Prompt:   request.Prompt,
			Images:   append([]string(nil), request.Images...),
			ImgURL:   imgURL,
			VideoURL: request.Video,
		},
		Parameters: parameters,
	}, nil
}

func validateKlingV3Request(request *KlingV3SubmitRequest) error {
	input := &request.Input
	parameters := &request.Parameters
	input.FirstFrameURL = strings.TrimSpace(input.FirstFrameURL)
	input.ImgURL = strings.TrimSpace(input.ImgURL)
	input.NegativePrompt = strings.TrimSpace(input.NegativePrompt)
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.VideoURL = strings.TrimSpace(input.VideoURL)
	parameters.AspectRatio = strings.TrimSpace(parameters.AspectRatio)
	parameters.CharacterOrientation = strings.TrimSpace(parameters.CharacterOrientation)
	externalTaskID := parameters.ExternalTaskID
	parameters.ExternalTaskID = strings.TrimSpace(parameters.ExternalTaskID)
	parameters.Image = strings.TrimSpace(parameters.Image)
	parameters.ImageTail = strings.TrimSpace(parameters.ImageTail)
	parameters.KeepOriginalSound = strings.TrimSpace(parameters.KeepOriginalSound)
	parameters.KlingV3Type = KlingV3GenerationType(strings.TrimSpace(string(parameters.KlingV3Type)))
	parameters.Mode = strings.TrimSpace(parameters.Mode)
	parameters.ShotType = strings.TrimSpace(parameters.ShotType)
	parameters.Sound = strings.TrimSpace(parameters.Sound)

	if utf8.RuneCountInString(input.Prompt) > 2500 {
		return errors.New("kling V3 prompt must not exceed 2500 characters")
	}
	if utf8.RuneCountInString(input.NegativePrompt) > 2500 {
		return errors.New("kling V3 negative_prompt must not exceed 2500 characters")
	}
	for i := range input.Images {
		input.Images[i] = strings.TrimSpace(input.Images[i])
		if input.Images[i] == "" {
			return fmt.Errorf("kling V3 images[%d] must not be empty", i)
		}
	}
	if input.FirstFrameURL != "" && !isAbsoluteURL(input.FirstFrameURL) {
		return errors.New("kling V3 first_frame_url must be an absolute URL")
	}
	if input.VideoURL != "" && !isAbsoluteURL(input.VideoURL) {
		return errors.New("kling V3 video_url must be an absolute URL")
	}
	if externalTaskID != "" && parameters.ExternalTaskID == "" {
		return errors.New("kling V3 external_task_id must not be blank")
	}
	if parameters.AspectRatio != "" && !oneOf(parameters.AspectRatio, "16:9", "9:16", "1:1") {
		return errors.New("kling V3 aspect_ratio must be 16:9, 9:16, or 1:1")
	}
	if parameters.CharacterOrientation != "" && !oneOf(parameters.CharacterOrientation, "image", "video") {
		return errors.New("kling V3 character_orientation must be image or video")
	}
	if parameters.KeepOriginalSound != "" && !oneOf(parameters.KeepOriginalSound, KeepOriginalSoundYes, KeepOriginalSoundNo) {
		return errors.New("kling V3 keep_original_sound must be yes or no")
	}
	if parameters.KlingV3Type != "" && !oneOf(string(parameters.KlingV3Type),
		string(KlingV3TypeT2V), string(KlingV3TypeI2V), string(KlingV3TypeMotionControl)) {
		return errors.New("kling V3 kling_v3_type must be t2v, i2v, or motion_control")
	}
	if parameters.Mode != "" && !oneOf(parameters.Mode, "std", "pro") {
		return errors.New("kling V3 mode must be std or pro")
	}
	if parameters.ShotType != "" && parameters.ShotType != "customize" {
		return errors.New("kling V3 shot_type must be customize")
	}
	if parameters.Sound != "" && !oneOf(parameters.Sound, "on", "off") {
		return errors.New("kling V3 sound must be on or off")
	}
	if parameters.ImageTail != "" && parameters.Image == "" {
		return errors.New("kling V3 image_tail requires image")
	}

	hasFirstFrame := input.FirstFrameURL != "" || len(input.Images) > 0 || input.ImgURL != "" || parameters.Image != ""
	generationType := parameters.KlingV3Type
	if generationType == "" {
		switch {
		case input.VideoURL != "":
			generationType = KlingV3TypeMotionControl
		case hasFirstFrame:
			generationType = KlingV3TypeI2V
		default:
			generationType = KlingV3TypeT2V
		}
	}

	switch generationType {
	case KlingV3TypeT2V:
		if input.VideoURL != "" || hasFirstFrame {
			return errors.New("kling V3 t2v does not accept image or video input")
		}
	case KlingV3TypeI2V:
		if input.VideoURL != "" {
			return errors.New("kling V3 i2v does not accept video_url")
		}
		if !hasFirstFrame {
			return errors.New("kling V3 i2v requires a first-frame image")
		}
	case KlingV3TypeMotionControl:
		if input.VideoURL == "" || input.ImgURL == "" {
			return errors.New("kling V3 motion_control requires input.video_url and input.img_url")
		}
		if parameters.CharacterOrientation == "" {
			return errors.New("kling V3 motion_control requires character_orientation")
		}
	}

	if parameters.Duration != 0 {
		if generationType == KlingV3TypeMotionControl {
			if parameters.Duration != 5 && parameters.Duration != 10 {
				return errors.New("kling V3 motion_control duration must be 5 or 10 seconds")
			}
		} else if parameters.Duration < 3 || parameters.Duration > 15 {
			return errors.New("kling V3 duration must be between 3 and 15 seconds")
		}
	}

	if parameters.MultiShot {
		if parameters.ShotType != "customize" {
			return errors.New("kling V3 multi_shot requires shot_type=customize")
		}
		if len(parameters.MultiPrompt) < 1 || len(parameters.MultiPrompt) > 6 {
			return errors.New("kling V3 multi_prompt must contain 1 to 6 shots")
		}
		if err := validateKlingV3MultiPrompt(parameters.MultiPrompt, parameters.Duration); err != nil {
			return err
		}
	} else if generationType != KlingV3TypeMotionControl && input.Prompt == "" {
		return errors.New("kling V3 prompt is required unless multi_shot or motion_control is enabled")
	}
	return nil
}

func validateKlingV3MultiPrompt(prompts []KlingV3MultiPromptItem, duration int) error {
	expectedDuration := duration
	if expectedDuration == 0 {
		expectedDuration = 5
	}
	totalDuration := 0
	seenIndexes := make(map[int]struct{}, len(prompts))
	for i := range prompts {
		prompt := &prompts[i]
		prompt.Duration = strings.TrimSpace(prompt.Duration)
		prompt.Prompt = strings.TrimSpace(prompt.Prompt)
		if prompt.Index < 1 {
			return fmt.Errorf("kling V3 multi_prompt[%d].index must be at least 1", i)
		}
		if _, exists := seenIndexes[prompt.Index]; exists {
			return fmt.Errorf("kling V3 multi_prompt index %d is duplicated", prompt.Index)
		}
		seenIndexes[prompt.Index] = struct{}{}
		if prompt.Prompt == "" || utf8.RuneCountInString(prompt.Prompt) > 512 {
			return fmt.Errorf("kling V3 multi_prompt[%d].prompt must contain 1 to 512 characters", i)
		}
		seconds, err := strconv.Atoi(prompt.Duration)
		if err != nil || seconds < 1 || seconds > expectedDuration {
			return fmt.Errorf("kling V3 multi_prompt[%d].duration must be an integer from 1 to %d", i, expectedDuration)
		}
		totalDuration += seconds
	}
	if totalDuration != expectedDuration {
		return fmt.Errorf("kling V3 multi_prompt durations total %d, want %d", totalDuration, expectedDuration)
	}
	return nil
}
