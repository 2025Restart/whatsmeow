// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import (
	"fmt"
	mathRand "math/rand"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/store"
)

// GenerateFingerprint 根据地区生成设备指纹
// regionCode: 地区代码，如 "IN", "BR"，空字符串使用默认配置
func GenerateFingerprint(regionCode string) *store.DeviceFingerprint {
	config := GetRegionConfig(regionCode)
	if config == nil {
		// 降级到默认配置
		config = defaultRegion
	}
	if config == nil {
		// 如果连默认配置都没有，返回空指纹（应该不会发生）
		return &store.DeviceFingerprint{}
	}

	return generateFromConfig(config)
}

// GenerateRandomFingerprint 生成随机设备指纹（向后兼容）
// 使用默认配置
func GenerateRandomFingerprint() *store.DeviceFingerprint {
	return GenerateFingerprint("")
}

// generateFromConfig 根据配置生成指纹
func generateFromConfig(config *RegionConfig) *store.DeviceFingerprint {
	// 1. 选择语言和国家
	lang, country := selectLanguageAndCountry(config)

	// 2. 选择平台（根据分布权重）
	platformDist := selectPlatform(config.DeviceDistribution)

	// 3. 选择制造商和设备型号
	manufacturer, device, modelType, board := selectDevice(config, platformDist)

	// 4. 选择 OS 版本
	osVersion := selectOSVersion(platformDist)

	// 5. 生成构建号（根据平台类型）
	osBuildNumber := generateBuildNumber(platformDist)

	// 6. 选择移动网络（MCC/MNC）
	mcc, mnc := selectMobileNetwork(config, country)

	// 7. 构建指纹
	return buildFingerprint(
		manufacturer, device, modelType, board,
		osVersion, osBuildNumber, mcc, mnc,
		lang, country, platformDist,
	)
}

// selectLanguageAndCountry 选择语言和国家
func selectLanguageAndCountry(config *RegionConfig) (lang, country string) {
	if len(config.Languages) == 0 {
		return "en", "US" // 默认值
	}

	// 根据权重随机选择语言
	r := mathRand.Float64()
	var cumWeight float64
	for _, langConfig := range config.Languages {
		cumWeight += langConfig.Weight
		if r <= cumWeight {
			lang = langConfig.Code
			// 从该语言对应的国家中随机选择一个
			if len(langConfig.Countries) > 0 {
				country = langConfig.Countries[mathRand.Intn(len(langConfig.Countries))]
			} else {
				country = "US" // 默认
			}
			return
		}
	}

	// 如果权重总和不足1，选择第一个
	lang = config.Languages[0].Code
	if len(config.Languages[0].Countries) > 0 {
		country = config.Languages[0].Countries[0]
	} else {
		country = "US"
	}
	return
}

// selectPlatform 根据权重选择平台
func selectPlatform(distributions []PlatformDistribution) PlatformDistribution {
	if len(distributions) == 0 {
		// 默认平台：Google Chrome（Web 浏览器）
		return PlatformDistribution{
			Platform:     waWa6.ClientPayload_UserAgent_WEB,
			PlatformType: waCompanionReg.DeviceProps_CHROME,
			DeviceType:   waWa6.ClientPayload_UserAgent_DESKTOP,
			OSName:       "Windows",
			OSVersions:   []string{"10", "11"},
		}
	}

	r := mathRand.Float64()
	var cumWeight float64
	for _, dist := range distributions {
		cumWeight += dist.Weight
		if r <= cumWeight {
			return dist
		}
	}

	// 如果权重总和不足1，返回第一个
	return distributions[0]
}

// selectDevice 选择制造商和设备型号
func selectDevice(config *RegionConfig, platformDist PlatformDistribution) (manufacturer, device, modelType, board string) {
	// 根据平台类型选择合适的制造商
	var manufacturers []string
	switch platformDist.PlatformType {
	case waCompanionReg.DeviceProps_ANDROID_PHONE, waCompanionReg.DeviceProps_ANDROID_TABLET:
		manufacturers = []string{"Samsung", "Xiaomi", "OnePlus", "Motorola", "Realme"}
	case waCompanionReg.DeviceProps_IOS_PHONE, waCompanionReg.DeviceProps_IPAD:
		manufacturers = []string{"Apple"}
	case waCompanionReg.DeviceProps_CATALINA:
		// macOS 平台，只选择 Apple 设备
		manufacturers = []string{"Apple"}
	case waCompanionReg.DeviceProps_CHROME, waCompanionReg.DeviceProps_FIREFOX, waCompanionReg.DeviceProps_EDGE:
		// Windows Web 平台，只选择 Windows 设备（排除 Apple）
		manufacturers = []string{"Microsoft", "Dell", "HP", "Lenovo", "Acer", "ASUS"}
	case waCompanionReg.DeviceProps_DESKTOP:
		// Linux 平台，选择 Linux 兼容设备（排除 Apple，因为 Mac 很少运行 Linux）
		manufacturers = []string{"Microsoft", "Dell", "HP", "Lenovo", "Acer", "ASUS"}
	default:
		// 其他平台，根据 OSName 判断
		if platformDist.OSName == "macOS" {
			manufacturers = []string{"Apple"}
		} else if platformDist.OSName == "Windows" {
			manufacturers = []string{"Microsoft", "Dell", "HP", "Lenovo", "Acer", "ASUS"}
		} else {
			// Linux 或其他
			manufacturers = []string{"Microsoft", "Dell", "HP", "Lenovo", "Acer", "ASUS"}
		}
	}

	// 从配置的设备型号池中选择
	manufacturer = manufacturers[mathRand.Intn(len(manufacturers))]
	if models, ok := config.DeviceModels[manufacturer]; ok && len(models) > 0 {
		model := models[mathRand.Intn(len(models))]
		device = model.Model
		modelType = model.ModelType
		board = model.Board
	} else {
		// 使用默认值
		device = "Unknown"
		modelType = ""
		board = ""
	}

	return
}

// selectOSVersion 选择 OS 版本
func selectOSVersion(platformDist PlatformDistribution) string {
	if len(platformDist.OSVersions) > 0 {
		return platformDist.OSVersions[mathRand.Intn(len(platformDist.OSVersions))]
	}
	return "1.0.0" // 默认版本
}

// selectMobileNetwork 选择移动网络（MCC/MNC）
func selectMobileNetwork(config *RegionConfig, country string) (mcc, mnc string) {
	if len(config.MobileNetworks) == 0 {
		// 使用默认 MCC/MNC
		return getDefaultMCCByCountry(country), "001"
	}

	// 根据权重随机选择
	r := mathRand.Float64()
	var cumWeight float64
	for _, net := range config.MobileNetworks {
		cumWeight += net.Weight
		if r <= cumWeight {
			return net.MCC, net.MNC
		}
	}

	// 如果权重总和不足1，返回第一个
	return config.MobileNetworks[0].MCC, config.MobileNetworks[0].MNC
}

// buildFingerprint 构建设备指纹
func buildFingerprint(
	manufacturer, device, modelType, board,
	osVersion, osBuildNumber, mcc, mnc,
	lang, country string,
	platformDist PlatformDistribution,
) *store.DeviceFingerprint {
	// 对于 Web 平台，使用浏览器名称；对于其他平台，使用操作系统名称
	devicePropsOs := getDevicePropsOs(platformDist)

	return &store.DeviceFingerprint{
		Manufacturer:       manufacturer,
		Device:             device,
		DeviceModelType:    modelType,
		OsVersion:          osVersion,
		OsBuildNumber:      osBuildNumber,
		Mcc:                mcc,
		Mnc:                mnc,
		LocaleLanguage:     lang,
		LocaleCountry:      country,
		Platform:           &platformDist.Platform,
		AppVersion:         store.GetWAVersion().ProtoAppVersion(), // 使用底层 API 获取最新版本
		DeviceType:         &platformDist.DeviceType,
		DeviceBoard:        board,
		DevicePropsOs:      devicePropsOs,
		DevicePropsVersion: generateDevicePropsVersion(platformDist),
		PlatformType:       &platformDist.PlatformType,
	}
}

// getDevicePropsOs 根据平台类型返回 DeviceProps.Os 的值
// 注意：当 PlatformType 是浏览器类型（CHROME, FIREFOX 等）时，WhatsApp 会自动添加浏览器名称前缀
// 因此 Os 字段应该只包含操作系统名称，避免重复显示
// 对于非浏览器平台，返回操作系统名称
func getDevicePropsOs(platformDist PlatformDistribution) string {
	// 获取操作系统显示名称
	osDisplayName := getOSDisplayName(platformDist.OSName)

	switch platformDist.PlatformType {
	case waCompanionReg.DeviceProps_CHROME,
		waCompanionReg.DeviceProps_FIREFOX,
		waCompanionReg.DeviceProps_EDGE,
		waCompanionReg.DeviceProps_SAFARI,
		waCompanionReg.DeviceProps_OPERA,
		waCompanionReg.DeviceProps_IE,
		waCompanionReg.DeviceProps_CATALINA,
		waCompanionReg.DeviceProps_DESKTOP:
		// 浏览器平台：只返回操作系统名称，WhatsApp 会根据 PlatformType 自动添加浏览器名称
		return osDisplayName
	default:
		// 对于非 Web 平台（Android, iOS 等），使用操作系统名称
		// 但如果是 Web 平台但未匹配到浏览器类型，也返回操作系统名称
		if platformDist.Platform == waWa6.ClientPayload_UserAgent_WEB {
			return osDisplayName
		}
		return platformDist.OSName
	}
}

// getOSDisplayName 将操作系统名称转换为显示名称
func getOSDisplayName(osName string) string {
	switch osName {
	case "Windows":
		return "Windows"
	case "macOS":
		return "Mac OS"
	case "Linux":
		return "Linux"
	case "Android":
		return "Android"
	case "iOS":
		return "iOS"
	default:
		// 如果未匹配，返回原名称或默认值
		if osName != "" {
			return osName
		}
		return "Windows" // 默认值
	}
}

// generateBuildNumber 生成构建号
func generateBuildNumber(platformDist PlatformDistribution) string {
	switch platformDist.PlatformType {
	case waCompanionReg.DeviceProps_ANDROID_PHONE, waCompanionReg.DeviceProps_ANDROID_TABLET:
		return generateAndroidBuildNumber(platformDist.OSVersions)
	case waCompanionReg.DeviceProps_IOS_PHONE, waCompanionReg.DeviceProps_IPAD:
		return generateIOSBuildNumber(platformDist.OSVersions)
	default:
		// 默认格式
		major := mathRand.Intn(5) + 20
		minor := mathRand.Intn(10)
		patch := mathRand.Intn(100)
		return fmt.Sprintf("%d%02d%02d", major, minor, patch)
	}
}

// generateAndroidBuildNumber 生成 Android 构建号
// 格式示例：TP1A.220624.014 (Android 13), UQ1A.231205.015 (Android 14)
func generateAndroidBuildNumber(osVersions []string) string {
	if len(osVersions) == 0 {
		return "TP1A.220624.014"
	}

	// 根据 OS 版本选择构建号前缀
	version := osVersions[mathRand.Intn(len(osVersions))]
	var prefix string
	switch version {
	case "11":
		prefix = "RQ"
	case "12":
		prefix = "SP"
	case "13":
		prefix = "TP"
	case "14":
		prefix = "UQ"
	default:
		prefix = "TP" // 默认 Android 13
	}

	// 生成日期部分（YYMMDD）
	year := 22 + mathRand.Intn(3) // 22-24
	month := 1 + mathRand.Intn(12)
	day := 1 + mathRand.Intn(28)

	// 生成修订号（3位数字）
	revision := mathRand.Intn(1000)

	// Android 构建号格式：TP1A.220624.014
	// 格式：[RTSU]Q[1-3][A-Z].[YYMMDD].[3位数字]
	// 字母部分通常是 A，但也可能是其他字母（B, C 等）
	buildLetter := byte('A' + mathRand.Intn(3)) // A, B, 或 C
	buildNumber := mathRand.Intn(3) + 1         // 1, 2, 或 3

	return fmt.Sprintf("%s%d%c.%02d%02d%02d.%03d", prefix, buildNumber, buildLetter, year, month, day, revision)
}

// generateIOSBuildNumber 生成 iOS/macOS 构建号
// 格式示例：20G95 (macOS 12.6.1), 21G115 (macOS 13.5.2)
func generateIOSBuildNumber(osVersions []string) string {
	if len(osVersions) == 0 {
		return "20G95"
	}

	version := osVersions[mathRand.Intn(len(osVersions))]
	var major int
	switch version {
	case "15":
		major = 20
	case "16":
		major = 21
	case "17":
		major = 22
	default:
		major = 21 // 默认 iOS 16
	}

	// 生成小版本和修订号
	minor := mathRand.Intn(10) + 1  // 1-10
	patch := mathRand.Intn(200) + 1 // 1-200

	return fmt.Sprintf("%dG%d", major, minor*10+patch)
}

// generateDevicePropsVersion 生成 DeviceProps 版本
// 根据 OS 版本生成对应的版本号
func generateDevicePropsVersion(platformDist PlatformDistribution) *waCompanionReg.DeviceProps_AppVersion {
	if len(platformDist.OSVersions) == 0 {
		return &waCompanionReg.DeviceProps_AppVersion{
			Primary:   proto.Uint32(13),
			Secondary: proto.Uint32(5),
			Tertiary:  proto.Uint32(2),
		}
	}

	osVersion := platformDist.OSVersions[mathRand.Intn(len(platformDist.OSVersions))]

	switch platformDist.PlatformType {
	case waCompanionReg.DeviceProps_ANDROID_PHONE, waCompanionReg.DeviceProps_ANDROID_TABLET:
		// Android 版本：11, 12, 13, 14 -> DeviceProps: {11, 0, 0}, {12, 0, 0}, etc.
		version := parseVersion(osVersion)
		return &waCompanionReg.DeviceProps_AppVersion{
			Primary:   proto.Uint32(version),
			Secondary: proto.Uint32(uint32(mathRand.Intn(10))),
			Tertiary:  proto.Uint32(uint32(mathRand.Intn(10))),
		}
	case waCompanionReg.DeviceProps_IOS_PHONE, waCompanionReg.DeviceProps_IPAD:
		// iOS 版本：15.x, 16.x, 17.x -> DeviceProps: {15, x, y}, {16, x, y}, etc.
		version := parseVersion(osVersion)
		return &waCompanionReg.DeviceProps_AppVersion{
			Primary:   proto.Uint32(version),
			Secondary: proto.Uint32(uint32(mathRand.Intn(10))),
			Tertiary:  proto.Uint32(uint32(mathRand.Intn(10))),
		}
	default:
		return &waCompanionReg.DeviceProps_AppVersion{
			Primary:   proto.Uint32(13),
			Secondary: proto.Uint32(5),
			Tertiary:  proto.Uint32(2),
		}
	}
}

// parseVersion 解析版本号字符串为数字
func parseVersion(version string) uint32 {
	var v uint32
	fmt.Sscanf(version, "%d", &v)
	if v == 0 {
		return 13 // 默认
	}
	return v
}

// getDefaultMCCByCountry 根据国家获取默认 MCC
func getDefaultMCCByCountry(country string) string {
	mccMap := map[string]string{
		"US": "310", "GB": "234", "CA": "302", "AU": "505", "NZ": "530",
		"CN": "460", "TW": "466", "HK": "454", "SG": "525",
		"ES": "214", "MX": "334", "AR": "722", "CO": "732",
		"FR": "208", "BE": "206", "CH": "228",
		"DE": "262", "AT": "232",
		"JP": "440", "KR": "450",
		"IN": "404", // 印度
		"BR": "724", // 巴西
	}
	if mcc, ok := mccMap[country]; ok {
		return mcc
	}
	return "310" // 默认 US
}
