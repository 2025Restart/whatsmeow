// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import (
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
)

// RegionConfig 地区配置
type RegionConfig struct {
	// 地区标识
	Code string // "IN", "BR", "US"
	Name string // "India", "Brazil", "United States"

	// 语言配置
	Languages []LanguageConfig // 支持的语言列表及权重

	// 设备分布（按平台）
	DeviceDistribution []PlatformDistribution

	// 移动网络配置
	MobileNetworks []MobileNetworkConfig // MCC/MNC 列表

	// 设备型号池（按制造商）
	DeviceModels map[string][]DeviceModelConfig
}

// LanguageConfig 语言配置
type LanguageConfig struct {
	Code      string   // "hi", "en", "pt"
	Weight    float64  // 权重（用于随机选择，总和应为 1.0）
	Countries []string // 对应的国家代码
}

// PlatformDistribution 平台分布
type PlatformDistribution struct {
	Platform     waWa6.ClientPayload_UserAgent_Platform
	PlatformType waCompanionReg.DeviceProps_PlatformType
	DeviceType   waWa6.ClientPayload_UserAgent_DeviceType
	Weight       float64 // 权重（如 Android 70%, iOS 30%，总和应为 1.0）
	OSName       string  // "Android", "iOS", "macOS", "Windows"
	OSVersions   []string // 该平台的 OS 版本列表
}

// MobileNetworkConfig 移动网络配置
type MobileNetworkConfig struct {
	MCC          string
	MNC          string
	OperatorName string
	Weight       float64 // 运营商市场份额（总和应为 1.0）
}

// DeviceModelConfig 设备型号配置
type DeviceModelConfig struct {
	Model       string
	ModelType   string // 设备型号类型
	Board       string
	BuildNumber string // 可选，为空则随机生成
}
