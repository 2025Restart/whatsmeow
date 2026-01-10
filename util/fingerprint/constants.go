// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import "go.mau.fi/whatsmeow/store"

// 设备指纹相关常量（从 store 包导入，避免循环依赖）
var (
	// DefaultBrowserName 默认浏览器名称
	DefaultBrowserName = store.DefaultBrowserName

	// DefaultOSName 默认操作系统名称（用于显示）
	DefaultOSName = store.DefaultOSName

	// DefaultDevicePropsOs 默认 DeviceProps.Os 值
	DefaultDevicePropsOs = store.DefaultDevicePropsOs
)

// GetDefaultDevicePropsOs 获取默认 DeviceProps.Os 值
// 注意：当 PlatformType 是浏览器类型时，WhatsApp 会自动添加浏览器名称前缀
// 因此 Os 字段应该只包含操作系统名称
// 格式：操作系统名称
func GetDefaultDevicePropsOs() string {
	return store.DefaultDevicePropsOs
}

// FormatDevicePropsOs 已废弃：不再使用此函数
// 当 PlatformType 是浏览器类型时，Os 字段应该只包含操作系统名称
// WhatsApp 会根据 PlatformType 自动添加浏览器名称前缀
// 保留此函数仅用于向后兼容，但不应再调用
// Deprecated: 使用 getOSDisplayName 直接返回操作系统名称
func FormatDevicePropsOs(browserName, osName string) string {
	osDisplayName := getOSDisplayNameForFormat(osName)
	return osDisplayName // 只返回操作系统名称，不包含浏览器名称
}

// getOSDisplayNameForFormat 将操作系统名称转换为显示名称（用于格式化）
func getOSDisplayNameForFormat(osName string) string {
	switch osName {
	case "Windows":
		return "Windows"
	case "macOS":
		return "Mac OS" // WhatsApp 显示格式
	case "Linux":
		return "Linux"
	case "Android":
		return "Android"
	case "iOS":
		return "iOS"
	default:
		if osName != "" {
			return osName
		}
		return "Windows" // 默认值
	}
}
