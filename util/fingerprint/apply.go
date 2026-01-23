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
// regionCode: 可选，地区代码（如 "IN", "BR"），用于设置兜底的 LocaleCountry 和 MCC/MNC
func ApplyFingerprint(payload *waWa6.ClientPayload, fp *store.DeviceFingerprint, regionCode ...string) {
	if payload == nil {
		return
	}

	// 确定使用的地区代码（优先使用传入的，否则从指纹中推断）
	var effectiveRegionCode string
	if len(regionCode) > 0 && regionCode[0] != "" {
		effectiveRegionCode = regionCode[0]
	} else if fp != nil && fp.LocaleCountry != "" {
		effectiveRegionCode = fp.LocaleCountry
	}

	// 获取地区配置（用于兜底值）
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

	// 如果指纹为空，仍然需要设置必填字段的兜底值（根据地区配置）
	if fp == nil {
		// 确保 UserAgent 存在
		if payload.UserAgent == nil {
			payload.UserAgent = &waWa6.ClientPayload_UserAgent{}
		}

		// 根据地区配置设置兜底值，避免 sanitizeClientPayload 硬编码为印度
		if effectiveRegionCode != "" && regionConfig != nil {
			// 设置 LocaleCountry
			if payload.UserAgent.LocaleCountryIso31661Alpha2 == nil {
				payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(effectiveRegionCode)
			}

			// 设置 MCC/MNC（使用地区配置的第一个）
			if len(regionConfig.MobileNetworks) > 0 {
				if payload.UserAgent.Mcc == nil {
					payload.UserAgent.Mcc = proto.String(regionConfig.MobileNetworks[0].MCC)
				}
				if payload.UserAgent.Mnc == nil {
					payload.UserAgent.Mnc = proto.String(regionConfig.MobileNetworks[0].MNC)
				}
			}

			// 设置语言（使用地区配置的第一个语言）
			if len(regionConfig.Languages) > 0 && payload.UserAgent.LocaleLanguageIso6391 == nil {
				payload.UserAgent.LocaleLanguageIso6391 = proto.String(regionConfig.Languages[0].Code)
			}
		}

		// 确保 WEB 平台特征正确
		if payload.UserAgent.GetPlatform() == waWa6.ClientPayload_UserAgent_WEB {
			payload.UserAgent.OsBuildNumber = nil
			payload.UserAgent.DeviceBoard = nil
			if payload.UserAgent.Device == nil {
				payload.UserAgent.Device = proto.String("Desktop")
			}
		}

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

	// 2. 强制地区特征对齐（根据配置的地区设置兜底值）
	if fp.LocaleLanguage != "" {
		payload.UserAgent.LocaleLanguageIso6391 = proto.String(fp.LocaleLanguage)
	} else if payload.UserAgent.LocaleLanguageIso6391 == nil {
		// 兜底：使用地区配置的第一个语言
		if regionConfig != nil && len(regionConfig.Languages) > 0 {
			payload.UserAgent.LocaleLanguageIso6391 = proto.String(regionConfig.Languages[0].Code)
		} else {
			payload.UserAgent.LocaleLanguageIso6391 = proto.String("en") // 默认值
		}
	}

	// 根据配置的地区设置兜底的 LocaleCountry 和 MCC/MNC
	// 优先级：指纹中的值 > 地区配置的默认值 > MCC 推断
	// 如果 LocaleCountry 为空，使用地区配置的默认值
	if fp.LocaleCountry == "" && effectiveRegionCode != "" {
		fp.LocaleCountry = effectiveRegionCode
	}

	// MCC/MNC 相互推断逻辑
	// 1. 如果只有 MCC，推断 MNC
	if fp.Mcc != "" && fp.Mnc == "" {
		// 优先使用地区配置的 MNC
		if regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
			// 找到匹配的 MCC
			for _, net := range regionConfig.MobileNetworks {
				if net.MCC == fp.Mcc {
					fp.Mnc = net.MNC
					break
				}
			}
		}
		// 如果还没找到，使用查找表推断
		if fp.Mnc == "" {
			fp.Mnc = InferMNCFromMCC(fp.Mcc)
		}
	}

	// 2. 如果只有 MNC，推断 MCC（需要国家代码）
	if fp.Mnc != "" && fp.Mcc == "" {
		// MNC=000 是固定宽带，无法从 MNC 推断 MCC，直接使用兜底地区
		if fp.Mnc == "000" {
			if regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
				fp.Mcc = regionConfig.MobileNetworks[0].MCC
			}
		} else {
			countryForInference := fp.LocaleCountry
			if countryForInference == "" && effectiveRegionCode != "" {
				countryForInference = effectiveRegionCode
			}
			if countryForInference != "" {
				fp.Mcc = InferMCCFromMNC(fp.Mnc, countryForInference)
			}
			// 如果还是没找到，使用地区配置
			if fp.Mcc == "" && regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
				// 找到匹配的 MNC
				for _, net := range regionConfig.MobileNetworks {
					if net.MNC == fp.Mnc {
						fp.Mcc = net.MCC
						break
					}
				}
				// 如果还是没找到匹配的 MNC，使用兜底地区的 MCC
				if fp.Mcc == "" {
					fp.Mcc = regionConfig.MobileNetworks[0].MCC
				}
			}
		}
	}

	// 3. 如果 MCC 为空，使用地区配置的默认 MCC/MNC
	// 注意：如果 MNC=000（固定宽带），不会被覆盖（因为只在 fp.Mnc == "" 时设置）
	if fp.Mcc == "" && regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
		fp.Mcc = regionConfig.MobileNetworks[0].MCC
		if fp.Mnc == "" {
			fp.Mnc = regionConfig.MobileNetworks[0].MNC
		}
	}

	// 4. 验证 MCC/MNC 组合有效性
	if fp.Mcc != "" && fp.Mnc != "" {
		// MNC=000 是固定宽带的合法值，不进行修正
		if fp.Mnc != "000" && !ValidateMCCMNC(fp.Mcc, fp.Mnc) {
			// 如果组合无效，尝试修正
			if inferredMNC := InferMNCFromMCC(fp.Mcc); inferredMNC != "" {
				fp.Mnc = inferredMNC
			} else if fp.LocaleCountry != "" {
				if inferredMCC := InferMCCFromMNC(fp.Mnc, fp.LocaleCountry); inferredMCC != "" {
					fp.Mcc = inferredMCC
				}
			}
		}
	}

	// 5. 如果 MCC 有值但 LocaleCountry 为空或不一致，根据 MCC 推断
	// 确保 MCC 和 LocaleCountry 始终保持一致（避免 rva/frc 封控）
	if fp.Mcc != "" && (fp.LocaleCountry == "" || !isMCCCountryMatch(fp.Mcc, fp.LocaleCountry)) {
		inferredCountry := getCountryByMCC(fp.Mcc)
		if inferredCountry != "" {
			oldCountry := fp.LocaleCountry
			fp.LocaleCountry = inferredCountry
			if oldCountry != "" && oldCountry != inferredCountry {
				// 记录同步日志（用于调试）
				// 注意：这里没有 logger，如果需要日志，应该在调用方添加
			}
		}
	}

	// 设置 LocaleCountry 和 MCC/MNC（根据指纹或地区配置）
	if fp.LocaleCountry != "" {
		payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(fp.LocaleCountry)
		countryCode := fp.LocaleCountry
		switch countryCode {
		case "IN":
			// 印度：确保 MCC 为 404 或 405
			if fp.Mcc == "" || (fp.Mcc != "404" && fp.Mcc != "405") {
				payload.UserAgent.Mcc = proto.String("404")
				// MNC=000 是固定宽带的合法值，必须保留
				if fp.Mnc == "000" {
					payload.UserAgent.Mnc = proto.String("000")
				} else {
					// 使用推断函数获取常见的 MNC
					if inferredMNC := InferMNCFromMCC("404"); inferredMNC != "" {
						payload.UserAgent.Mnc = proto.String(inferredMNC)
					} else {
						payload.UserAgent.Mnc = proto.String("01")
					}
				}
			} else {
				payload.UserAgent.Mcc = proto.String(fp.Mcc)
				if fp.Mnc != "" {
					payload.UserAgent.Mnc = proto.String(fp.Mnc)
				} else {
					// 使用推断函数
					if inferredMNC := InferMNCFromMCC(fp.Mcc); inferredMNC != "" {
						payload.UserAgent.Mnc = proto.String(inferredMNC)
					} else {
						payload.UserAgent.Mnc = proto.String("01")
					}
				}
			}
		case "BR":
			// 巴西：确保 MCC 为 724
			if fp.Mcc == "" || fp.Mcc != "724" {
				payload.UserAgent.Mcc = proto.String("724")
				// MNC=000 是固定宽带的合法值，必须保留
				if fp.Mnc == "000" {
					payload.UserAgent.Mnc = proto.String("000")
				} else {
					// 使用推断函数获取常见的 MNC
					if inferredMNC := InferMNCFromMCC("724"); inferredMNC != "" {
						payload.UserAgent.Mnc = proto.String(inferredMNC)
					} else {
						payload.UserAgent.Mnc = proto.String("02")
					}
				}
			} else {
				payload.UserAgent.Mcc = proto.String(fp.Mcc)
				if fp.Mnc != "" {
					payload.UserAgent.Mnc = proto.String(fp.Mnc)
				} else {
					// 使用推断函数
					if inferredMNC := InferMNCFromMCC(fp.Mcc); inferredMNC != "" {
						payload.UserAgent.Mnc = proto.String(inferredMNC)
					} else {
						payload.UserAgent.Mnc = proto.String("02")
					}
				}
			}
		default:
			// 其他国家：直接应用指纹的 MCC/MNC
			if fp.Mcc != "" {
				payload.UserAgent.Mcc = proto.String(fp.Mcc)
			}
			if fp.Mnc != "" {
				payload.UserAgent.Mnc = proto.String(fp.Mnc)
			}
		}
	} else if effectiveRegionCode != "" {
		// 如果 LocaleCountry 为空，使用地区配置
		payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(effectiveRegionCode)
		// 根据地区设置 MCC/MNC
		// MNC=000 是固定宽带的合法值，必须保留
		if fp.Mnc == "000" {
			// 保留 MNC=000，只设置 MCC
			if fp.Mcc != "" {
				payload.UserAgent.Mcc = proto.String(fp.Mcc)
			} else if regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
				payload.UserAgent.Mcc = proto.String(regionConfig.MobileNetworks[0].MCC)
			} else {
				switch effectiveRegionCode {
				case "IN":
					payload.UserAgent.Mcc = proto.String("404")
				case "BR":
					payload.UserAgent.Mcc = proto.String("724")
				}
			}
			payload.UserAgent.Mnc = proto.String("000")
		} else {
			// 正常设置 MCC/MNC
			if regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
				payload.UserAgent.Mcc = proto.String(regionConfig.MobileNetworks[0].MCC)
				payload.UserAgent.Mnc = proto.String(regionConfig.MobileNetworks[0].MNC)
			} else {
				// 根据地区推断
				switch effectiveRegionCode {
				case "IN":
					payload.UserAgent.Mcc = proto.String("404")
					payload.UserAgent.Mnc = proto.String("01")
				case "BR":
					payload.UserAgent.Mcc = proto.String("724")
					payload.UserAgent.Mnc = proto.String("02")
				}
			}
		}
	}

	// 最终兜底：确保 MCC/MNC 和 LocaleCountry 都被设置
	if payload.UserAgent.Mcc == nil || payload.UserAgent.Mnc == nil {
		if regionConfig != nil && len(regionConfig.MobileNetworks) > 0 {
			if payload.UserAgent.Mcc == nil {
				payload.UserAgent.Mcc = proto.String(regionConfig.MobileNetworks[0].MCC)
			}
			if payload.UserAgent.Mnc == nil {
				// 如果 MCC 已设置，尝试推断 MNC
				if payload.UserAgent.Mcc != nil {
					if inferredMNC := InferMNCFromMCC(payload.UserAgent.GetMcc()); inferredMNC != "" {
						payload.UserAgent.Mnc = proto.String(inferredMNC)
					} else {
						payload.UserAgent.Mnc = proto.String(regionConfig.MobileNetworks[0].MNC)
					}
				} else {
					payload.UserAgent.Mnc = proto.String(regionConfig.MobileNetworks[0].MNC)
				}
			}
		} else if effectiveRegionCode != "" {
			switch effectiveRegionCode {
			case "IN":
				if payload.UserAgent.Mcc == nil {
					payload.UserAgent.Mcc = proto.String("404")
				}
				if payload.UserAgent.Mnc == nil {
					if inferredMNC := InferMNCFromMCC(payload.UserAgent.GetMcc()); inferredMNC != "" {
						payload.UserAgent.Mnc = proto.String(inferredMNC)
					} else {
						payload.UserAgent.Mnc = proto.String("01")
					}
				}
			case "BR":
				if payload.UserAgent.Mcc == nil {
					payload.UserAgent.Mcc = proto.String("724")
				}
				if payload.UserAgent.Mnc == nil {
					if inferredMNC := InferMNCFromMCC(payload.UserAgent.GetMcc()); inferredMNC != "" {
						payload.UserAgent.Mnc = proto.String(inferredMNC)
					} else {
						payload.UserAgent.Mnc = proto.String("02")
					}
				}
			}
		}
	}

	// 如果只有 MCC 或只有 MNC，使用推断函数补充
	if payload.UserAgent.Mcc != nil && payload.UserAgent.Mnc == nil {
		if inferredMNC := InferMNCFromMCC(payload.UserAgent.GetMcc()); inferredMNC != "" {
			payload.UserAgent.Mnc = proto.String(inferredMNC)
		}
	}
	if payload.UserAgent.Mnc != nil && payload.UserAgent.Mcc == nil {
		countryForInference := ""
		if payload.UserAgent.LocaleCountryIso31661Alpha2 != nil {
			countryForInference = payload.UserAgent.GetLocaleCountryIso31661Alpha2()
		} else if effectiveRegionCode != "" {
			countryForInference = effectiveRegionCode
		}
		if countryForInference != "" {
			if inferredMCC := InferMCCFromMNC(payload.UserAgent.GetMnc(), countryForInference); inferredMCC != "" {
				payload.UserAgent.Mcc = proto.String(inferredMCC)
			}
		}
	}

	if payload.UserAgent.LocaleCountryIso31661Alpha2 == nil {
		if payload.UserAgent.Mcc != nil {
			inferredCountry := getCountryByMCC(payload.UserAgent.GetMcc())
			if inferredCountry != "" {
				payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(inferredCountry)
			} else if effectiveRegionCode != "" {
				payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(effectiveRegionCode)
			}
		} else if effectiveRegionCode != "" {
			payload.UserAgent.LocaleCountryIso31661Alpha2 = proto.String(effectiveRegionCode)
		}
	}

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

// getCountryByMCC 根据 MCC 返回对应的国家代码（包内使用）
func getCountryByMCC(mcc string) string {
	return GetCountryByMCC(mcc)
}

// GetCountryByMCC 根据 MCC 返回对应的国家代码（导出函数）
func GetCountryByMCC(mcc string) string {
	switch mcc {
	case "404", "405":
		return "IN"
	case "724":
		return "BR"
	default:
		return ""
	}
}

// isMCCCountryMatch 检查 MCC 是否与国家代码匹配
func isMCCCountryMatch(mcc, countryCode string) bool {
	if mcc == "" || countryCode == "" {
		return true // 无法判断时不触发强制修正
	}
	// 常见的国家-MCC 映射
	switch countryCode {
	case "IN":
		return mcc == "404" || mcc == "405"
	case "BR":
		return mcc == "724"
	case "US":
		return strings.HasPrefix(mcc, "310") || strings.HasPrefix(mcc, "311") || strings.HasPrefix(mcc, "312") || strings.HasPrefix(mcc, "313") || strings.HasPrefix(mcc, "316")
	case "GB":
		return mcc == "234" || mcc == "235"
	case "RU":
		return mcc == "250"
	case "ID":
		return mcc == "510"
	default:
		// 兜底逻辑：如果前两位/三位数字不符合大洲范围，可能不匹配
		// 这种通用检查避免大部分 frc/rva 封控
		return true
	}
}
