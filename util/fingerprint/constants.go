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
// 格式：浏览器名称 (操作系统名称)
func GetDefaultDevicePropsOs() string {
	return store.DefaultDevicePropsOs
}

// FormatDevicePropsOs 格式化 DeviceProps.Os
// browserName: 浏览器名称，如 "Google Chrome", "Firefox"
// osName: 操作系统名称，如 "Windows", "macOS", "Linux"
// 返回格式：浏览器名称 (操作系统显示名称)
// 示例：FormatDevicePropsOs("Google Chrome", "macOS") -> "Google Chrome (Mac OS)"
func FormatDevicePropsOs(browserName, osName string) string {
	osDisplayName := getOSDisplayNameForFormat(osName)
	return browserName + " (" + osDisplayName + ")"
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
