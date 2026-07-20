package model

// VideoPackage stores downloadable application packages and their aggregate
// installation/download/device statistics.
type VideoPackage struct {
	ID             uint64   `json:"id" gorm:"primaryKey;autoIncrement"`
	PackageName    string   `json:"package_name" gorm:"size:128;not null;index;comment:package name"`
	PackageCode    string   `json:"package_code" gorm:"size:128;not null;uniqueIndex:uk_video_package_code_version,priority:1;index;comment:package identifier"`
	PackageVersion string   `json:"package_version" gorm:"size:64;not null;uniqueIndex:uk_video_package_code_version,priority:2;index;comment:package version"`
	Language       string   `json:"language" gorm:"size:16;not null;default:zh-CN;index;comment:default API response language"`
	SystemTypes    []string `json:"system_types" gorm:"type:text;serializer:json;comment:supported system types"`
	DownloadURL    string   `json:"download_url" gorm:"size:1024;not null;comment:package download URL"`
	InstallCount   uint64   `json:"install_count" gorm:"not null;default:0;comment:installation count"`
	DownloadCount  uint64   `json:"download_count" gorm:"not null;default:0;comment:download count"`
	DeviceCount    uint64   `json:"device_count" gorm:"not null;default:0;comment:device count"`
	Description    string   `json:"description" gorm:"type:text;comment:package description"`
	Sort           int      `json:"sort" gorm:"not null;default:0;index;comment:sort order"`
	Status         int8     `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	BaseModel
}

func (VideoPackage) TableName() string {
	return "video_package"
}
