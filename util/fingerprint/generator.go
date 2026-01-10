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

	// 5. 生成构建号
	osBuildNumber := generateBuildNumber()

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
		// 默认平台
		return PlatformDistribution{
			Platform:     waWa6.ClientPayload_UserAgent_WEB,
			PlatformType: waCompanionReg.DeviceProps_DESKTOP,
			DeviceType:   waWa6.ClientPayload_UserAgent_DESKTOP,
			OSName:       "Linux",
			OSVersions:   []string{"5.15.0"},
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
	case waCompanionReg.DeviceProps_DESKTOP:
		manufacturers = []string{"Apple", "Microsoft", "Dell", "HP", "Lenovo"}
	default:
		manufacturers = []string{"Samsung", "Apple", "Microsoft"}
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
	return &store.DeviceFingerprint{
		Manufacturer:    manufacturer,
		Device:          device,
		DeviceModelType: modelType,
		OsVersion:       osVersion,
		OsBuildNumber:   osBuildNumber,
		Mcc:             mcc,
		Mnc:             mnc,
		LocaleLanguage:  lang,
		LocaleCountry:   country,
		Platform:        &platformDist.Platform,
		AppVersion: &waWa6.ClientPayload_UserAgent_AppVersion{
			Primary:   proto.Uint32(uint32(mathRand.Intn(3) + 2)),
			Secondary: proto.Uint32(uint32(mathRand.Intn(3000) + 2000)),
			Tertiary:  proto.Uint32(uint32(mathRand.Intn(1000000000))),
		},
		DeviceType:    &platformDist.DeviceType,
		DeviceBoard:   board,
		DevicePropsOs: platformDist.OSName,
		DevicePropsVersion: &waCompanionReg.DeviceProps_AppVersion{
			Primary:   proto.Uint32(uint32(mathRand.Intn(5) + 10)),
			Secondary: proto.Uint32(uint32(mathRand.Intn(10))),
			Tertiary:  proto.Uint32(uint32(mathRand.Intn(10))),
		},
		PlatformType: &platformDist.PlatformType,
	}
}

// generateBuildNumber 生成构建号
func generateBuildNumber() string {
	major := mathRand.Intn(5) + 20
	minor := mathRand.Intn(10)
	patch := mathRand.Intn(100)
	return fmt.Sprintf("%d%02d%02d", major, minor, patch)
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
