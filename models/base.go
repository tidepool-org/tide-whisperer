package models

import (
	"time"
)

type Model interface {
	GetType() string
}

type Base struct {
	Time              time.Time  `mapstructure:"time" pg:"time,type:timestamptz" json:"time,omitempty"`

	Type              string     `mapstructure:"type" pg:"-" json:"type,omitempty"`

	CreatedTime       time.Time  `mapstructure:"createdTime" pg:"created_time,type:timestamptz" json:"-"`
	ModifiedTime      time.Time  `mapstructure:"modifiedTime" pg:"modified_time,type:timestamptz" json:"-"`
	DeviceTime        time.Time  `mapstructure:"deviceTime" pg:"device_time,type:timestamptz" json:"deviceTime,omitempty"`

	DeviceId          string   `mapstructure:"deviceId,omitempty" pg:"device_id" json:"deviceId,omitempty"`
	Id                string   `mapstructure:"id,omitempty" pg:"id" json:"id,omitempty"`

	Timezone          string   `mapstructure:"timezone,omitempty" pg:"timezone" json:"timezone,omitempty"`
	TimezoneOffset    int64    `mapstructure:"timezoneOffset,omitempty" pg:"timezone_offset" json:"timezoneOffset,omitempty"`
	ClockDriftOffset  int64    `mapstructure:"clockDriftOffset,omitempty" pg:"clock_drift_offset" json:"clockDriftOffset,omitempty"`
	ConversionOffset  int64    `mapstructure:"conversionOffset,omitempty" pg:"conversion_offset" json:"conversionOffset,omitempty"`

	UploadId          string   `mapstructure:"uploadId,omitempty" pg:"upload_id" json:"uploadId,omitempty"`
	UserId            string   `mapstructure:"_userId,omitempty" pg:"user_id" json:"-"`

	PayloadJson       string     `pg:"payload" json:"payload"`

	Revision          int64   `mapstructure:"revision,omitempty" pg:"revision" json:"-"`
}

func (b *Base) GetType() string {
	return b.Type
}

func (b *Base) GetUserId() string {
	return b.UserId
}


