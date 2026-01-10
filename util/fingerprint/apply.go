// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import (
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/store"
)

// ApplyFingerprint 应用设备指纹到 ClientPayload
func ApplyFingerprint(payload *waWa6.ClientPayload, fp *store.DeviceFingerprint) {
	if payload == nil || fp == nil {
		return
	}

	// 确保 UserAgent 存在
	if payload.UserAgent == nil {
		payload.UserAgent = &waWa6.ClientPayload_UserAgent{}
	}

	// 应用 UserAgent 字段
	if fp.Manufacturer != "" {
		payload.UserAgent.Manufacturer = proto.String(fp.Manufacturer)
	}
	if fp.Device != "" {
		payload.UserAgent.Device = proto.String(fp.Device)
	}
	if fp.DeviceModelType != "" {
		payload.UserAgent.DeviceModelType = proto.String(fp.DeviceModelType)
	}
	if fp.OsVersion != "" {
		payload.UserAgent.OsVersion = proto.String(fp.OsVersion)
	}
	if fp.OsBuildNumber != "" {
		payload.UserAgent.OsBuildNumber = proto.String(fp.OsBuildNumber)
	}
	if fp.Mcc != "" {
		payload.UserAgent.Mcc = proto.String(fp.Mcc)
	}
	if fp.Mnc != "" {
		payload.UserAgent.Mnc = proto.String(fp.Mnc)
	}
	if fp.LocaleLanguage != "" {
		payload.UserAgent.LocaleLanguageIso6391 = proto.String(fp.LocaleLanguage)
	}
	if fp.LocaleCountry != "" {
		payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(fp.LocaleCountry)
	}
	if fp.Platform != nil {
		payload.UserAgent.Platform = fp.Platform
	}
	if fp.AppVersion != nil {
		payload.UserAgent.AppVersion = fp.AppVersion
	}
	if fp.DeviceType != nil {
		payload.UserAgent.DeviceType = fp.DeviceType
	}
	if fp.DeviceBoard != "" {
		payload.UserAgent.DeviceBoard = proto.String(fp.DeviceBoard)
	}

	// 应用 DeviceProps（在 DevicePairingData 中）
	// 注意：需要合并，不能完全替换，保留其他重要字段
	if fp.DevicePropsOs != "" {
		// 如果 DevicePairingData 不存在，在注册时会创建
		// 这里只处理已存在的情况
		if payload.DevicePairingData != nil {
			// 解析现有的 DeviceProps
			var existingProps waCompanionReg.DeviceProps
			if len(payload.DevicePairingData.DeviceProps) > 0 {
				_ = proto.Unmarshal(payload.DevicePairingData.DeviceProps, &existingProps)
			}

			// 创建新的 DeviceProps，合并字段
			deviceProps := &waCompanionReg.DeviceProps{
				Os: proto.String(fp.DevicePropsOs),
			}

			if fp.DevicePropsVersion != nil {
				deviceProps.Version = fp.DevicePropsVersion
			}

			if fp.PlatformType != nil {
				deviceProps.PlatformType = fp.PlatformType
			}

			// 保留其他重要字段（从现有或使用默认值）
			if existingProps.HistorySyncConfig != nil {
				deviceProps.HistorySyncConfig = existingProps.HistorySyncConfig
			} else {
				// 使用默认值
				deviceProps.HistorySyncConfig = &waCompanionReg.DeviceProps_HistorySyncConfig{
					StorageQuotaMb:                           proto.Uint32(10240),
					InlineInitialPayloadInE2EeMsg:            proto.Bool(true),
					SupportCallLogHistory:                    proto.Bool(false),
					SupportBotUserAgentChatHistory:           proto.Bool(true),
					SupportCagReactionsAndPolls:              proto.Bool(true),
					SupportBizHostedMsg:                      proto.Bool(true),
					SupportRecentSyncChunkMessageCountTuning: proto.Bool(true),
					SupportHostedGroupMsg:                    proto.Bool(true),
					SupportFbidBotChatHistory:                proto.Bool(true),
					SupportMessageAssociation:                proto.Bool(true),
					SupportGroupHistory:                      proto.Bool(false),
				}
			}
			if existingProps.RequireFullSync != nil {
				deviceProps.RequireFullSync = existingProps.RequireFullSync
			} else {
				deviceProps.RequireFullSync = proto.Bool(false)
			}

			// 序列化并设置
			devicePropsBytes, _ := proto.Marshal(deviceProps)
			payload.DevicePairingData.DeviceProps = devicePropsBytes
		}
	}
}
