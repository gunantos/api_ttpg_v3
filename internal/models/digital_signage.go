package models

import "time"

type DigitalSignageDevice struct {
	ID                  uint       `gorm:"primaryKey;column:id;type:int(11);autoIncrement" json:"id"`
	DeviceID            string     `gorm:"column:device_id;size:50;not null;uniqueIndex:uq_ds_devices_device_id" json:"device_id"`
	Name                string     `gorm:"column:name;size:100" json:"name"`
	Location            string     `gorm:"column:location;size:150" json:"location"`
	Resolution          string     `gorm:"column:resolution;size:30" json:"resolution"`
	Orientation         string     `gorm:"column:orientation;size:20;default:landscape" json:"orientation"`
	Volume              int        `gorm:"column:volume;type:int(11);default:80" json:"volume"`
	Brightness          int        `gorm:"column:brightness;type:int(11);default:80" json:"brightness"`
	Status              string     `gorm:"column:status;size:20;default:pending" json:"status"`
	LastSeenAt          *time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`
	ConfigVersion       string     `gorm:"column:config_version;size:50" json:"config_version"`
	TokenHash           string     `gorm:"column:token_hash;size:128" json:"-"`
	TokenPrefix         string     `gorm:"column:token_prefix;size:16" json:"token_prefix,omitempty"`
	TokenRotatedAt      *time.Time `gorm:"column:token_rotated_at" json:"token_rotated_at,omitempty"`
	AppVersion          string     `gorm:"column:app_version;size:50" json:"app_version"`
	IPAddress           string     `gorm:"column:ip_address;size:64" json:"ip_address"`
	LastSyncAt          *time.Time `gorm:"column:last_sync_at" json:"last_sync_at"`
	LastMetricsJSON     string     `gorm:"column:last_metrics_json;type:mediumtext" json:"last_metrics,omitempty"`
	CpuPercent          int        `gorm:"column:cpu_percent;type:int(11);default:0" json:"cpu_percent"`
	MemoryPercent       int        `gorm:"column:memory_percent;type:int(11);default:0" json:"memory_percent"`
	DiskPercent         int        `gorm:"column:disk_percent;type:int(11);default:0" json:"disk_percent"`
	LastMetricAt        *time.Time `gorm:"column:last_metric_at" json:"last_metric_at"`
	CurrentMediaID      *uint      `gorm:"column:current_media_id;type:int(11)" json:"current_media_id"`
	CurrentPlaylistID   *uint      `gorm:"column:current_playlist_id;type:int(11)" json:"current_playlist_id"`
	CurrentLayoutID     *uint      `gorm:"column:current_layout_id;type:int(11)" json:"current_layout_id"`
	MainLayoutID        *uint      `gorm:"column:main_layout_id;type:int(11);index:idx_ds_devices_main_layout" json:"main_layout_id"`
	ScreenshotURL       string     `gorm:"column:screenshot_url;size:500" json:"screenshot_url,omitempty"`
	Timezone            string     `gorm:"column:timezone;size:64;default:Asia/Jakarta" json:"timezone"`
	CacheLimitMB        int        `gorm:"column:cache_limit_mb;type:int(11);default:2048" json:"cache_limit_mb"`
	SettingsJSON        string     `gorm:"column:settings_json;type:text" json:"settings_json,omitempty"`
	PublishedRevisionID *uint64    `gorm:"column:published_revision_id" json:"published_revision_id"`
	ApprovedAt          *time.Time `gorm:"column:approved_at" json:"approved_at"`
	LastOfflineEventAt  *time.Time `gorm:"column:last_offline_event_at" json:"last_offline_event_at"`
	CreatedAt           time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (DigitalSignageDevice) TableName() string { return "ds_devices" }
