// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package store

import (
	"context"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/types"
)

// DeviceFingerprint 设备指纹信息
type DeviceFingerprint struct {
	// UserAgent 字段
	Manufacturer    string
	Device          string
	DeviceModelType string
	OsVersion       string
	OsBuildNumber   string
	Mcc             string
	Mnc             string
	LocaleLanguage  string
	LocaleCountry   string
	Platform        *waWa6.ClientPayload_UserAgent_Platform
	AppVersion      *waWa6.ClientPayload_UserAgent_AppVersion
	DeviceType      *waWa6.ClientPayload_UserAgent_DeviceType
	DeviceBoard     string

	// DeviceProps 字段
	DevicePropsOs      string
	DevicePropsVersion *waCompanionReg.DeviceProps_AppVersion
	PlatformType       *waCompanionReg.DeviceProps_PlatformType
}

// FingerprintStore 设备指纹存储接口
type FingerprintStore interface {
	GetFingerprint(ctx context.Context, jid types.JID) (*DeviceFingerprint, error)
	PutFingerprint(ctx context.Context, jid types.JID, fp *DeviceFingerprint) error
}
