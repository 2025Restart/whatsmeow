// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import (
	"math/rand"
	"strings"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/store"
)

// ApplyFingerprint 应用设备指纹到 ClientPayload
// regionCode: 可选，地区代码（如 "IN", "BR"），仅用于设备指纹生成（设备型号选择等），不用于地理信息
// 注意：地理信息（Country/Language/MCC/MNC）应由业务层通过 SetUserLoginGeoInfo 和 SetCarrierInfo 传入
func ApplyFingerprint(payload *waWa6.ClientPayload, fp *store.DeviceFingerprint, regionCode ...string) {
	if payload == nil {
		return
	}

	// 确定使用的地区代码（仅用于设备指纹生成，如设备型号选择）
	var effectiveRegionCode string
	if len(regionCode) > 0 && regionCode[0] != "" {
		effectiveRegionCode = regionCode[0]
	} else if fp != nil && fp.LocaleCountry != "" {
		effectiveRegionCode = fp.LocaleCountry
	}

	// 获取地区配置（仅用于设备指纹生成，不用于地理信息）
	var regionConfig *RegionConfig
	if effectiveRegionCode != "" {
		regionConfig = GetRegionConfig(effectiveRegionCode)
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

	// 确保 UserAgent 存在
	if payload.UserAgent == nil {
		payload.UserAgent = &waWa6.ClientPayload_UserAgent{}
	}

	// 如果指纹为空，仅设置最基本的字段
	if fp == nil {
		// 确保 WEB 平台特征正确
		if payload.UserAgent.GetPlatform() == waWa6.ClientPayload_UserAgent_WEB {
			payload.UserAgent.OsBuildNumber = nil
			payload.UserAgent.DeviceBoard = nil
			if payload.UserAgent.Device == nil {
				payload.UserAgent.Device = proto.String("Desktop")
			}
		}
		// 统一设置 MCC/MNC（在函数末尾统一处理）
		applyCarrierInfo(payload, nil, regionConfig)
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

	// 确保必填字段不为空（兜底逻辑）
	if payload.UserAgent.OsVersion == nil || payload.UserAgent.GetOsVersion() == "" {
		if fp.OsVersion != "" {
			payload.UserAgent.OsVersion = proto.String(fp.OsVersion)
		} else {
			payload.UserAgent.OsVersion = proto.String("10.0.0") // 默认值
		}
	}
	if payload.UserAgent.Manufacturer == nil || payload.UserAgent.GetManufacturer() == "" || payload.UserAgent.GetManufacturer() == "Unknown" {
		if fp.Manufacturer != "" {
			payload.UserAgent.Manufacturer = proto.String(fp.Manufacturer)
		} else {
			payload.UserAgent.Manufacturer = proto.String("Microsoft") // 默认值
		}
	}
	if payload.UserAgent.Device == nil || payload.UserAgent.GetDevice() == "" {
		if fp.Device != "" {
			payload.UserAgent.Device = proto.String(fp.Device)
		} else {
			payload.UserAgent.Device = proto.String("Desktop") // 默认值
		}
	}

	// 2. 设置语言（业务层已传入，直接应用）
	if fp.LocaleLanguage != "" {
		payload.UserAgent.LocaleLanguageIso6391 = proto.String(fp.LocaleLanguage)
	} else if payload.UserAgent.LocaleLanguageIso6391 == nil {
		// 兜底：仅设置默认值，不根据地区配置推断
		payload.UserAgent.LocaleLanguageIso6391 = proto.String("en") // 默认值
	}

	// 设置 LocaleCountry（业务层已传入，直接应用）
	if fp.LocaleCountry != "" {
		payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(fp.LocaleCountry)
	}

	// 统一设置 MCC/MNC（在函数末尾统一处理）
	applyCarrierInfo(payload, fp, regionConfig)

	// 3. WEB 平台特征深度清洗 (解决 vll 报错)
	isWeb := payload.UserAgent.GetPlatform() == waWa6.ClientPayload_UserAgent_WEB
	if isWeb {
		// 强制清理移动端残留字段（彻底清理，避免 vll 封控）
		payload.UserAgent.OsBuildNumber = nil
		payload.UserAgent.DeviceBoard = nil
		payload.UserAgent.DeviceModelType = nil                  // 移动端字段
		payload.UserAgent.OsVersion = proto.String(fp.OsVersion) // 确保有值但无 BuildNumber

		// 强制对齐桌面特征
		payload.UserAgent.Device = proto.String("Desktop")
		if payload.UserAgent.Manufacturer == nil || *payload.UserAgent.Manufacturer == "Unknown" {
			// 根据 OS 设置默认 Manufacturer
			if strings.Contains(strings.ToLower(fp.DevicePropsOs), "windows") {
				payload.UserAgent.Manufacturer = proto.String("Microsoft")
			} else if strings.Contains(strings.ToLower(fp.DevicePropsOs), "mac") {
				payload.UserAgent.Manufacturer = proto.String("Apple")
			}
		}

		// 确保 Platform 是 WEB
		if payload.UserAgent.Platform == nil {
			payload.UserAgent.Platform = waWa6.ClientPayload_UserAgent_WEB.Enum()
		}

		// 针对 Windows/macOS 强制设置通用制造商
		osName := ""
		if fp.DevicePropsOs != "" {
			osName = strings.ToLower(fp.DevicePropsOs)
		} else if payload.UserAgent.OsVersion != nil {
			// 从 OsVersion 推断
			osVer := strings.ToLower(payload.UserAgent.GetOsVersion())
			if strings.Contains(osVer, "windows") || strings.Contains(osVer, "10") || strings.Contains(osVer, "11") {
				osName = "windows"
			} else if strings.Contains(osVer, "mac") {
				osName = "mac"
			}
		}
		if strings.Contains(osName, "windows") {
			payload.UserAgent.Manufacturer = proto.String("Microsoft")
		} else if strings.Contains(osName, "mac") {
			payload.UserAgent.Manufacturer = proto.String("Apple")
		} else if payload.UserAgent.Manufacturer == nil || *payload.UserAgent.Manufacturer == "Unknown" {
			// 兜底：默认 Microsoft（Windows）
			payload.UserAgent.Manufacturer = proto.String("Microsoft")
		}
	} else {
		// 非 WEB 平台保留原有逻辑
		if fp.OsBuildNumber != "" {
			payload.UserAgent.OsBuildNumber = proto.String(fp.OsBuildNumber)
		}
		if fp.DeviceBoard != "" {
			payload.UserAgent.DeviceBoard = proto.String(fp.DeviceBoard)
		}
		if fp.DeviceModelType != "" {
			payload.UserAgent.DeviceModelType = proto.String(fp.DeviceModelType)
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

	// 5. 确保 WebInfo 正确设置（避免 lla 封控）
	if payload.WebInfo == nil {
		payload.WebInfo = &waWa6.ClientPayload_WebInfo{
			WebSubPlatform: waWa6.ClientPayload_WebInfo_WEB_BROWSER.Enum(),
		}
	} else if payload.WebInfo.WebSubPlatform == nil {
		// 确保 WebSubPlatform 不为空
		payload.WebInfo.WebSubPlatform = waWa6.ClientPayload_WebInfo_WEB_BROWSER.Enum()
	}

	// 6. 最终验证：确保所有必填字段都已设置（避免 atn/cln/lla/vll 封控）
	if payload.UserAgent != nil {
		// 确保 Platform 正确设置
		if payload.UserAgent.Platform == nil {
			payload.UserAgent.Platform = waWa6.ClientPayload_UserAgent_WEB.Enum()
		}

		// 确保 ReleaseChannel 设置
		if payload.UserAgent.ReleaseChannel == nil {
			payload.UserAgent.ReleaseChannel = waWa6.ClientPayload_UserAgent_RELEASE.Enum()
		}

		// 确保 AppVersion 设置
		if payload.UserAgent.AppVersion == nil {
			payload.UserAgent.AppVersion = store.GetWAVersion().ProtoAppVersion()
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

// applyCarrierInfo 统一设置 MCC/MNC
// 优先级：外界设置 > 地区配置（如果开启）> 兜底 "000"
func applyCarrierInfo(payload *waWa6.ClientPayload, fp *store.DeviceFingerprint, regionConfig *RegionConfig) {
	if payload == nil || payload.UserAgent == nil {
		return
	}

	// 设置 MCC：优先级 外界 > 地区配置 > "000"
	if payload.UserAgent.Mcc == nil {
		// 1. 优先使用外界设置（通过 fp.Mcc）
		if fp != nil && fp.Mcc != "" {
			payload.UserAgent.Mcc = proto.String(fp.Mcc)
		} else if regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
			// 2. 地区配置开启：使用地区配置（根据权重随机选择）
			payload.UserAgent.Mcc = proto.String(selectMCCByWeight(regionConfig.MobileNetworks))
		} else {
			// 3. 兜底：设置为 "000"
			payload.UserAgent.Mcc = proto.String("000")
		}
	}

	// 设置 MNC：优先级 外界 > 地区配置 > "000"
	if payload.UserAgent.Mnc == nil {
		// 1. 优先使用外界设置（通过 fp.Mnc）
		if fp != nil && fp.Mnc != "" {
			payload.UserAgent.Mnc = proto.String(fp.Mnc)
		} else if regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
			// 2. 地区配置开启：MNC 统一设置为 "000"（固定宽带）
			payload.UserAgent.Mnc = proto.String("000")
		} else {
			// 3. 兜底：设置为 "000"
			payload.UserAgent.Mnc = proto.String("000")
		}
	}
}

// selectMCCByWeight 根据权重随机选择 MCC
func selectMCCByWeight(networks []MobileNetworkConfig) string {
	if len(networks) == 0 {
		return ""
	}
	
	r := rand.Float64()
	var cumWeight float64
	for _, net := range networks {
		cumWeight += net.Weight
		if r <= cumWeight {
			return net.MCC
		}
	}
	
	// 如果权重总和不足1，返回第一个
	return networks[0].MCC
}

