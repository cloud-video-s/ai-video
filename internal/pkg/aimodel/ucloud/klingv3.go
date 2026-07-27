package ucloud

import "github.com/gin-gonic/gin"

const (
	klingV3Model = "kling-v3"
)

type T struct {
	Model string `json:"model"`
	Input struct {
	} `json:"input"`
	Parameters Parameters `json:"parameters"`
}

type Input struct {
	Prompt         string   `json:"prompt"`          //提示词
	FirstFrameUrl  string   `json:"first_frame_url"` //首帧图像 URL
	Images         []string `json:"images"`          //首帧图像输入
	ImgUrl         string   `json:"img_url"`         //参考图片url
	NegativePrompt string   `json:"negative_prompt"` //限制不期望内容的负向提示词
	VideoUrl       string   `json:"video_url"`       //参考视频 URL
}

type Parameters struct {
	KlingV3Type           string `json:"kling_v3_type"`
	Mode                  string `json:"mode"`
	AspectRatio           string `json:"aspect_ratio"`          //生成视频的宽高比 16:9、9:16、1:1
	character_orientation string `json:"character_orientation"` // 模式下角色面向方向 image、video
	Duration              int    `json:"duration"`              //生成视频的时长（秒） t2v 和 i2v 支持 3 到 15；motion_control 支持 5 或 10。
	external_task_id      string `json:"external_task_id"`      //客户端提供的任务 ID
	Sound                 string `json:"sound"`
}

func TaskSubmit(c *gin.Context, req interface{}) {

}
