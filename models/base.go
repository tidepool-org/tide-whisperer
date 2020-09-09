package models

import (
	"time"
)

type Model interface {
	GetType() string
}

type Base struct {
	Time              time.Time  `mapstructure:"time" pg:"time,type:timestamptz"`

	Type              string     `mapstructure:"type" pg:"-"`

	CreatedTime       time.Time  `mapstructure:"createdTime" pg:"created_time,type:timestamptz"`
	ModifiedTime      time.Time  `mapstructure:"modifiedTime" pg:"modified_time,type:timestamptz"`
	DeviceTime        time.Time  `mapstructure:"deviceTime" pg:"device_time,type:timestamptz"`

	DeviceId          string   `mapstructure:"deviceId,omitempty" pg:"device_id"`
	Id                string   `mapstructure:"id,omitempty" pg:"id"`

	Timezone          string   `mapstructure:"timezone,omitempty" pg:"timezone"`
	TimezoneOffset    int64    `mapstructure:"timezoneOffset,omitempty" pg:"timezone_offset"`
	ClockDriftOffset  int64    `mapstructure:"clockDriftOffset,omitempty" pg:"clock_drift_offset"`
	ConversionOffset  int64    `mapstructure:"conversionOffset,omitempty" pg:"conversion_offset"`

	UploadId          string   `mapstructure:"uploadId,omitempty" pg:"upload_id"`
	UserId            string   `mapstructure:"_userId,omitempty" pg:"user_id"`

	Revision          int64   `mapstructure:"revision,omitempty" pg:"revision"`
}

func (b *Base) GetType() string {
	return b.Type
}

func (b *Base) GetUserId() string {
	return b.UserId
}


