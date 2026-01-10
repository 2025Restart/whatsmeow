// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import (
	"strings"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/store"
)

// ApplyFingerprint 应用设备指纹到 ClientPayload
func ApplyFingerprint(payload *waWa6.ClientPayload, fp *store.DeviceFingerprint) {
	if payload == nil {
		return
	}

	// 即使指纹为空，也检查并覆盖非官方标识（三重保障）
	if payload.DevicePairingData != nil && len(payload.DevicePairingData.DeviceProps) > 0 {
		var existingProps waCompanionReg.DeviceProps
		if err := proto.Unmarshal(payload.DevicePairingData.DeviceProps, &existingProps); err == nil {
			if existingProps.Os != nil && *existingProps.Os == "whatsmeow" {
				// 强制覆盖非官方标识
				if fp != nil && fp.DevicePropsOs != "" {
					existingProps.Os = proto.String(fp.DevicePropsOs)
					// 同时设置 PlatformType 以显示图标
					if fp.PlatformType != nil {
						existingProps.PlatformType = fp.PlatformType
					} else {
						// 根据 Os 内容推断 PlatformType
						existingProps.PlatformType = inferPlatformTypeFromOs(fp.DevicePropsOs)
					}
				} else {
					existingProps.Os = proto.String(DefaultDevicePropsOs) // 使用统一管理的默认值
					// 默认使用 CHROME 以显示图标
					existingProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
				}
				// PC 浏览器特征：强制清理可能导致风控的移动端残留字段
				if payload.UserAgent != nil && (payload.WebInfo != nil || existingProps.GetPlatformType() == waCompanionReg.DeviceProps_CHROME) {
					// 官方 Web 通常不发送具体的 MCC/MNC，设为 000
					payload.UserAgent.Mcc = proto.String("000")
					payload.UserAgent.Mnc = proto.String("000")
					// L2: 强制对齐平台类型
					payload.UserAgent.Platform = waWa6.ClientPayload_UserAgent_WEB.Enum()
					// 强制设置桌面类型
					payload.UserAgent.Device = proto.String("Desktop")
				}
				devicePropsBytes, _ := proto.Marshal(&existingProps)
				payload.DevicePairingData.DeviceProps = devicePropsBytes
			}
		}
	}

	// 如果指纹为空，直接返回（已处理非官方标识）
	if fp == nil {
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
	// 强制应用 MCC/MNC（即使为空也要覆盖默认值 "000"）
	if fp.Mcc != "" {
		payload.UserAgent.Mcc = proto.String(fp.Mcc)
	}
	if fp.Mnc != "" {
		payload.UserAgent.Mnc = proto.String(fp.Mnc)
	}
	// 强制应用语言和国家（即使为空也要覆盖默认值）
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

			// 设置 PlatformType（用于显示浏览器图标）
			if fp.PlatformType != nil {
				deviceProps.PlatformType = fp.PlatformType
			} else {
				// 如果 PlatformType 未设置，根据 Os 内容推断
				deviceProps.PlatformType = inferPlatformTypeFromOs(fp.DevicePropsOs)
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

// inferPlatformTypeFromOs 根据 DeviceProps.Os 的内容推断 PlatformType
// 用于确保浏览器图标能正确显示
func inferPlatformTypeFromOs(osValue string) *waCompanionReg.DeviceProps_PlatformType {
	if osValue == "" {
		return waCompanionReg.DeviceProps_CHROME.Enum() // 默认 Chrome
	}

	// 根据浏览器名称推断 PlatformType
	osLower := strings.ToLower(osValue)
	if strings.Contains(osLower, "chrome") {
		return waCompanionReg.DeviceProps_CHROME.Enum()
	} else if strings.Contains(osLower, "firefox") {
		return waCompanionReg.DeviceProps_FIREFOX.Enum()
	} else if strings.Contains(osLower, "safari") {
		return waCompanionReg.DeviceProps_SAFARI.Enum()
	} else if strings.Contains(osLower, "edge") {
		return waCompanionReg.DeviceProps_EDGE.Enum()
	} else if strings.Contains(osLower, "opera") {
		return waCompanionReg.DeviceProps_OPERA.Enum()
	} else if strings.Contains(osLower, "internet explorer") || strings.Contains(osLower, "ie ") {
		return waCompanionReg.DeviceProps_IE.Enum()
	}

	// 默认返回 Chrome（因为大多数 Web 平台使用 Chrome）
	return waCompanionReg.DeviceProps_CHROME.Enum()
}
