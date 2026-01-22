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

// GetRegionByPhone 根据手机号前缀识别地区
func GetRegionByPhone(phone string) string {
	info := LookupPhoneRegion(phone)
	if info != nil {
		return info.RegionCode
	}
	return ""
}

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

				// 深度清理特征逻辑移至下方统一处理
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

	// 1. 应用基础设备字段
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
	if fp.Platform != nil {
		payload.UserAgent.Platform = fp.Platform
	}
	if fp.AppVersion != nil {
		payload.UserAgent.AppVersion = fp.AppVersion
	}
	if fp.DeviceType != nil {
		payload.UserAgent.DeviceType = fp.DeviceType
	}

	// 2. 强制地区特征对齐 (IN/BR 专项)
	if fp.LocaleLanguage != "" {
		payload.UserAgent.LocaleLanguageIso6391 = proto.String(fp.LocaleLanguage)
	}
	if fp.LocaleCountry != "" {
		payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(fp.LocaleCountry)

		// 针对特定国家强制修正 MCC/MNC，解决 odn 报错
		countryCode := fp.LocaleCountry
		switch countryCode {
		case "IN":
			// 印度：确保 MCC 为 404 或 405
			if fp.Mcc == "" || !strings.HasPrefix(fp.Mcc, "40") {
				payload.UserAgent.Mcc = proto.String("404")
				payload.UserAgent.Mnc = proto.String("01")
			} else if fp.Mcc != "" {
				// 如果指纹中有有效的 MCC（以 40 开头），直接应用
				payload.UserAgent.Mcc = proto.String(fp.Mcc)
				if fp.Mnc != "" {
					payload.UserAgent.Mnc = proto.String(fp.Mnc)
				} else {
					payload.UserAgent.Mnc = proto.String("01")
				}
			}
		case "BR":
			// 巴西：确保 MCC 为 724
			if fp.Mcc == "" || fp.Mcc != "724" {
				payload.UserAgent.Mcc = proto.String("724")
				payload.UserAgent.Mnc = proto.String("02")
			} else if fp.Mcc == "724" {
				// 如果指纹中的 MCC 正确，应用指纹的 MNC
				payload.UserAgent.Mcc = proto.String(fp.Mcc)
				if fp.Mnc != "" {
					payload.UserAgent.Mnc = proto.String(fp.Mnc)
				} else {
					payload.UserAgent.Mnc = proto.String("02")
				}
			}
		}
	}

	// 如果上面开关没命中，但指纹里有 MCC，则应用指纹的（覆盖已设置的值）
	// 注意：对于 IN/BR 国家，如果上面 switch 已经处理了，这里不会再次设置
	// 对于其他国家，或者上面 switch 未处理的情况，直接应用指纹的 MCC/MNC
	if fp.Mcc != "" && (fp.LocaleCountry == "" || (fp.LocaleCountry != "IN" && fp.LocaleCountry != "BR")) {
		payload.UserAgent.Mcc = proto.String(fp.Mcc)
	}
	if fp.Mnc != "" && (fp.LocaleCountry == "" || (fp.LocaleCountry != "IN" && fp.LocaleCountry != "BR")) {
		payload.UserAgent.Mnc = proto.String(fp.Mnc)
	}

	// 3. WEB 平台特征深度清洗 (解决 vll 报错)
	isWeb := payload.UserAgent.GetPlatform() == waWa6.ClientPayload_UserAgent_WEB
	if isWeb {
		// 强制清理移动端残留字段
		payload.UserAgent.OsBuildNumber = nil
		payload.UserAgent.DeviceBoard = nil

		// 强制对齐桌面特征
		payload.UserAgent.Device = proto.String("Desktop")

		// 针对 Windows/macOS 强制设置通用制造商
		osName := ""
		if fp.DevicePropsOs != "" {
			osName = strings.ToLower(fp.DevicePropsOs)
		}
		if strings.Contains(osName, "windows") {
			payload.UserAgent.Manufacturer = proto.String("Microsoft")
		} else if strings.Contains(osName, "mac") {
			payload.UserAgent.Manufacturer = proto.String("Apple")
		}
	} else {
		// 非 WEB 平台保留原有逻辑
		if fp.OsBuildNumber != "" {
			payload.UserAgent.OsBuildNumber = proto.String(fp.OsBuildNumber)
		}
		if fp.DeviceBoard != "" {
			payload.UserAgent.DeviceBoard = proto.String(fp.DeviceBoard)
		}
	}

	// 4. 应用 DeviceProps（在 DevicePairingData 中）
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
				// 使用默认值（与 store/clientpayload.go 保持一致）
				deviceProps.HistorySyncConfig = &waCompanionReg.DeviceProps_HistorySyncConfig{
					StorageQuotaMb:                proto.Uint32(10240),
					InlineInitialPayloadInE2EeMsg: proto.Bool(true),
					RecentSyncDaysLimit:           nil,
					// 不需要的功能默认关闭，最小化特征
					SupportCallLogHistory:                    proto.Bool(true),
					SupportBotUserAgentChatHistory:           proto.Bool(false),
					SupportCagReactionsAndPolls:              proto.Bool(false),
					SupportBizHostedMsg:                      proto.Bool(false),
					SupportRecentSyncChunkMessageCountTuning: proto.Bool(true),
					SupportHostedGroupMsg:                    proto.Bool(false),
					SupportFbidBotChatHistory:                proto.Bool(false),
					SupportAddOnHistorySyncMigration:         nil,
					SupportMessageAssociation:                proto.Bool(false),
					SupportGroupHistory:                      proto.Bool(false),
					OnDemandReady:                            nil,
					SupportGuestChat:                         nil,
					CompleteOnDemandReady:                    nil,
					ThumbnailSyncDaysLimit:                   nil,
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
