// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

var _ store.FingerprintStore = (*Container)(nil)

const (
	getFingerprintQuery = `
        SELECT manufacturer, device, device_model_type, os_version, os_build_number,
               mcc, mnc, locale_language, locale_country, platform,
               app_version_primary, app_version_secondary, app_version_tertiary,
               device_type, device_board,
               device_props_os, device_props_version_primary, device_props_version_secondary, device_props_version_tertiary,
               platform_type
        FROM whatsmeow_device_fingerprint WHERE jid=$1
    `

	putFingerprintQuery = `
        INSERT INTO whatsmeow_device_fingerprint (
            jid, manufacturer, device, device_model_type, os_version, os_build_number,
            mcc, mnc, locale_language, locale_country, platform,
            app_version_primary, app_version_secondary, app_version_tertiary,
            device_type, device_board,
            device_props_os, device_props_version_primary, device_props_version_secondary, device_props_version_tertiary,
            platform_type, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, CURRENT_TIMESTAMP)
        ON CONFLICT (jid) DO UPDATE SET
            manufacturer=excluded.manufacturer, device=excluded.device, device_model_type=excluded.device_model_type,
            os_version=excluded.os_version, os_build_number=excluded.os_build_number,
            mcc=excluded.mcc, mnc=excluded.mnc,
            locale_language=excluded.locale_language, locale_country=excluded.locale_country,
            platform=excluded.platform,
            app_version_primary=excluded.app_version_primary, app_version_secondary=excluded.app_version_secondary, app_version_tertiary=excluded.app_version_tertiary,
            device_type=excluded.device_type, device_board=excluded.device_board,
            device_props_os=excluded.device_props_os,
            device_props_version_primary=excluded.device_props_version_primary, device_props_version_secondary=excluded.device_props_version_secondary, device_props_version_tertiary=excluded.device_props_version_tertiary,
            platform_type=excluded.platform_type, updated_at=excluded.updated_at
    `
)

// GetFingerprint 获取设备指纹
func (c *Container) GetFingerprint(ctx context.Context, jid types.JID) (*store.DeviceFingerprint, error) {
	var fp store.DeviceFingerprint
	var platform, deviceType, platformType sql.NullInt32
	var appVerPrimary, appVerSecondary, appVerTertiary sql.NullInt32
	var devPropsVerPrimary, devPropsVerSecondary, devPropsVerTertiary sql.NullInt32

	err := c.db.QueryRow(ctx, getFingerprintQuery, jid.String()).Scan(
		&fp.Manufacturer, &fp.Device, &fp.DeviceModelType, &fp.OsVersion, &fp.OsBuildNumber,
		&fp.Mcc, &fp.Mnc, &fp.LocaleLanguage, &fp.LocaleCountry, &platform,
		&appVerPrimary, &appVerSecondary, &appVerTertiary,
		&deviceType, &fp.DeviceBoard,
		&fp.DevicePropsOs, &devPropsVerPrimary, &devPropsVerSecondary, &devPropsVerTertiary,
		&platformType,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprint: %w", err)
	}

	// 转换枚举值
	if platform.Valid {
		p := waWa6.ClientPayload_UserAgent_Platform(platform.Int32)
		fp.Platform = &p
	}
	if deviceType.Valid {
		dt := waWa6.ClientPayload_UserAgent_DeviceType(deviceType.Int32)
		fp.DeviceType = &dt
	}
	if platformType.Valid {
		pt := waCompanionReg.DeviceProps_PlatformType(platformType.Int32)
		fp.PlatformType = &pt
	}
	if appVerPrimary.Valid {
		fp.AppVersion = &waWa6.ClientPayload_UserAgent_AppVersion{
			Primary:   proto.Uint32(uint32(appVerPrimary.Int32)),
			Secondary: proto.Uint32(uint32(appVerSecondary.Int32)),
			Tertiary:  proto.Uint32(uint32(appVerTertiary.Int32)),
		}
	}
	if devPropsVerPrimary.Valid {
		fp.DevicePropsVersion = &waCompanionReg.DeviceProps_AppVersion{
			Primary:   proto.Uint32(uint32(devPropsVerPrimary.Int32)),
			Secondary: proto.Uint32(uint32(devPropsVerSecondary.Int32)),
			Tertiary:  proto.Uint32(uint32(devPropsVerTertiary.Int32)),
		}
	}

	return &fp, nil
}

// PutFingerprint 保存设备指纹
func (c *Container) PutFingerprint(ctx context.Context, jid types.JID, fp *store.DeviceFingerprint) error {
	var platform, deviceType, platformType sql.NullInt32
	var appVerPrimary, appVerSecondary, appVerTertiary sql.NullInt32
	var devPropsVerPrimary, devPropsVerSecondary, devPropsVerTertiary sql.NullInt32

	if fp.Platform != nil {
		platform = sql.NullInt32{Int32: int32(*fp.Platform), Valid: true}
	}
	if fp.DeviceType != nil {
		deviceType = sql.NullInt32{Int32: int32(*fp.DeviceType), Valid: true}
	}
	if fp.PlatformType != nil {
		platformType = sql.NullInt32{Int32: int32(*fp.PlatformType), Valid: true}
	}
	if fp.AppVersion != nil {
		appVerPrimary = sql.NullInt32{Int32: int32(fp.AppVersion.GetPrimary()), Valid: true}
		appVerSecondary = sql.NullInt32{Int32: int32(fp.AppVersion.GetSecondary()), Valid: true}
		appVerTertiary = sql.NullInt32{Int32: int32(fp.AppVersion.GetTertiary()), Valid: true}
	}
	if fp.DevicePropsVersion != nil {
		devPropsVerPrimary = sql.NullInt32{Int32: int32(fp.DevicePropsVersion.GetPrimary()), Valid: true}
		devPropsVerSecondary = sql.NullInt32{Int32: int32(fp.DevicePropsVersion.GetSecondary()), Valid: true}
		devPropsVerTertiary = sql.NullInt32{Int32: int32(fp.DevicePropsVersion.GetTertiary()), Valid: true}
	}

	_, err := c.db.Exec(ctx, putFingerprintQuery,
		jid.String(), fp.Manufacturer, fp.Device, fp.DeviceModelType, fp.OsVersion, fp.OsBuildNumber,
		fp.Mcc, fp.Mnc, fp.LocaleLanguage, fp.LocaleCountry, platform,
		appVerPrimary, appVerSecondary, appVerTertiary,
		deviceType, fp.DeviceBoard,
		fp.DevicePropsOs, devPropsVerPrimary, devPropsVerSecondary, devPropsVerTertiary,
		platformType,
	)
	if err != nil {
		return fmt.Errorf("failed to put fingerprint: %w", err)
	}
	return nil
}
