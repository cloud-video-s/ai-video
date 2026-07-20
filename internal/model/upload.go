package model

const (
	UploadUserUnknown int8 = 0
	UploadUserAdmin   int8 = 1
	UploadUserClient  int8 = 2

	UploadMediaImage = "image"
	UploadMediaVideo = "video"

	UploadStorageLocal     = "local"
	UploadStorageAliyunOSS = "aliyun_oss"
)

type VideoUpload struct {
	ID              uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	UploadID        string `json:"upload_id" gorm:"size:32;not null;uniqueIndex"`
	UserType        int8   `json:"user_type" gorm:"type:tinyint;not null;index:idx_video_upload_owner,priority:1;comment:用户类型 1=admin 2=客户端"`
	UserID          uint64 `json:"user_id" gorm:"not null;index:idx_video_upload_owner,priority:2;comment:用户ID"`
	MediaType       string `json:"media_type" gorm:"size:16;not null;index"`
	FileType        string `json:"file_type" gorm:"size:32;not null;index"`
	MIMEType        string `json:"mime_type" gorm:"size:128;not null"`
	OriginalName    string `json:"original_name" gorm:"size:255;not null"`
	FileSize        uint64 `json:"file_size" gorm:"not null"`
	StorageProvider string `json:"storage_provider" gorm:"size:32;not null;index"`
	FilePath        string `json:"file_path" gorm:"size:1024;not null"`
	FileURL         string `json:"file_url" gorm:"type:text;not null"`
	SHA256          string `json:"sha256" gorm:"size:64;not null;index"`
	BaseModel
}

func (VideoUpload) TableName() string { return "video_upload" }
