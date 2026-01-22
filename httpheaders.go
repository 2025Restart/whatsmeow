// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package whatsmeow

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/socket"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// generateUserAgent 根据指纹信息生成浏览器 User-Agent 字符串
func (cli *Client) generateUserAgent() string {
	fp := cli.getFingerprintInfo()
	
	// 如果没有指纹信息，使用默认值
	platformType := fp.PlatformType
	if platformType == 0 {
		platformType = waCompanionReg.DeviceProps_CHROME
	}
	osName := fp.DevicePropsOs
	if osName == "" {
		osName = "Windows"
	}
	osVersion := fp.OsVersion
	if osVersion == "" {
		osVersion = "10.0.0"
	}

	// 根据平台类型生成 User-Agent
	return generateBrowserUserAgent(platformType, osName, osVersion)
}

// fingerprintInfo 指纹信息结构
type fingerprintInfo struct {
	PlatformType  waCompanionReg.DeviceProps_PlatformType
	DevicePropsOs string
	OsVersion      string
	LocaleLanguage string
	LocaleCountry  string
}

// getFingerprintInfo 获取指纹信息（从数据库或临时指纹）
func (cli *Client) getFingerprintInfo() fingerprintInfo {
	var fp fingerprintInfo
	
	if container, ok := cli.Store.Container.(*sqlstore.Container); ok {
		jid := cli.Store.GetJID()
		if jid.IsEmpty() {
			// 配对阶段，尝试使用临时指纹
			cli.pendingFingerprintLock.RLock()
			if cli.pendingFingerprint != nil {
				if cli.pendingFingerprint.PlatformType != nil {
					fp.PlatformType = *cli.pendingFingerprint.PlatformType
				}
				fp.DevicePropsOs = cli.pendingFingerprint.DevicePropsOs
				fp.OsVersion = cli.pendingFingerprint.OsVersion
				fp.LocaleLanguage = cli.pendingFingerprint.LocaleLanguage
				fp.LocaleCountry = cli.pendingFingerprint.LocaleCountry
			}
			cli.pendingFingerprintLock.RUnlock()
		} else {
			// 已配对，从数据库读取指纹
			if dbFp, err := container.GetFingerprint(context.TODO(), jid); err == nil && dbFp != nil {
				if dbFp.PlatformType != nil {
					fp.PlatformType = *dbFp.PlatformType
				}
				fp.DevicePropsOs = dbFp.DevicePropsOs
				fp.OsVersion = dbFp.OsVersion
				fp.LocaleLanguage = dbFp.LocaleLanguage
				fp.LocaleCountry = dbFp.LocaleCountry
			}
		}
	}
	
	return fp
}

// generateBrowserUserAgent 根据浏览器类型和操作系统生成 User-Agent
func generateBrowserUserAgent(platformType waCompanionReg.DeviceProps_PlatformType, osName, osVersion string) string {
	osLower := strings.ToLower(osName)
	
	switch platformType {
	case waCompanionReg.DeviceProps_CHROME:
		// Chrome User-Agent 格式（使用较新的版本号）
		// 参考：Chrome 130+ 版本（2024-2025）
		osPart := getOSUserAgentPart(osName, osVersion)
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36", osPart)
	case waCompanionReg.DeviceProps_FIREFOX:
		// Firefox User-Agent 格式（使用较新的版本号）
		osPart := getOSUserAgentPart(osName, osVersion)
		return fmt.Sprintf("Mozilla/5.0 (%s; rv:132.0) Gecko/20100101 Firefox/132.0", osPart)
	case waCompanionReg.DeviceProps_EDGE:
		// Edge User-Agent 格式（Edge 版本通常与 Chrome 版本接近）
		osPart := getOSUserAgentPart(osName, osVersion)
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0", osPart)
	case waCompanionReg.DeviceProps_SAFARI, waCompanionReg.DeviceProps_CATALINA:
		// Safari User-Agent 格式（macOS）
		// 参考：Safari 18.x 版本（macOS Sequoia/Sonoma）
		if strings.Contains(osLower, "mac") {
			// macOS 版本格式：将 "12.7.6" 转换为 "12_7_6" 或 "12" 转换为 "12_0_0"
			macVersion := normalizeMacOSVersion(osVersion)
			// Safari 版本根据 macOS 版本调整（macOS 12+ 使用 Safari 15+, macOS 13+ 使用 Safari 16+, macOS 14+ 使用 Safari 17+）
			safariVersion := getSafariVersionForMacOS(osVersion)
			return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15", macVersion, safariVersion)
		}
		// 非 macOS 使用 Chrome 格式
		osPart := getOSUserAgentPart(osName, osVersion)
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36", osPart)
	case waCompanionReg.DeviceProps_OPERA:
		// Opera User-Agent 格式（Opera 版本通常比 Chrome 稍低）
		osPart := getOSUserAgentPart(osName, osVersion)
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 OPR/117.0.0.0", osPart)
	default:
		// 默认使用 Chrome
		osPart := getOSUserAgentPart(osName, osVersion)
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36", osPart)
	}
}

// getSafariVersionForMacOS 根据 macOS 版本返回对应的 Safari 版本
// macOS 12 (Monterey) -> Safari 15.x
// macOS 13 (Ventura) -> Safari 16.x
// macOS 14 (Sonoma) -> Safari 17.x
// macOS 15 (Sequoia) -> Safari 18.x
func getSafariVersionForMacOS(osVersion string) string {
	parts := strings.Split(osVersion, ".")
	if len(parts) == 0 {
		return "18.1" // 默认最新版本
	}
	majorVersion := parts[0]
	switch majorVersion {
	case "12":
		return "15.6.1"
	case "13":
		return "16.6"
	case "14":
		return "17.6"
	case "15":
		return "18.1"
	default:
		// 对于未知版本，根据主版本号推断
		if majorInt := parseInt(majorVersion); majorInt >= 15 {
			return "18.1"
		} else if majorInt >= 14 {
			return "17.6"
		} else if majorInt >= 13 {
			return "16.6"
		} else {
			return "15.6.1"
		}
	}
}

// parseInt 简单的字符串转整数（用于版本号解析）
func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			break
		}
	}
	return result
}

// normalizeMacOSVersion 标准化 macOS 版本格式
// 将 "12" 转换为 "12_0_0"，"12.7" 转换为 "12_7_0"，"12.7.6" 转换为 "12_7_6"
func normalizeMacOSVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) == 1 {
		// 只有主版本号，如 "12"
		return fmt.Sprintf("%s_0_0", parts[0])
	} else if len(parts) == 2 {
		// 主版本号和次版本号，如 "12.7"
		return fmt.Sprintf("%s_%s_0", parts[0], parts[1])
	} else {
		// 完整版本号，如 "12.7.6"
		return strings.Join(parts, "_")
	}
}

// getOSUserAgentPart 生成 User-Agent 中的操作系统部分
func getOSUserAgentPart(osName, osVersion string) string {
	osLower := strings.ToLower(osName)
	
	if strings.Contains(osLower, "windows") {
		// Windows NT 10.0; Win64; x64
		// 确保版本格式正确（如 "10" 或 "10.0" 都转换为 "10.0"）
		winVersion := normalizeWindowsVersion(osVersion)
		return fmt.Sprintf("Windows NT %s; Win64; x64", winVersion)
	} else if strings.Contains(osLower, "linux") {
		// X11; Linux x86_64
		return "X11; Linux x86_64"
	} else {
		// 默认 Windows
		return "Windows NT 10.0; Win64; x64"
	}
}

// normalizeWindowsVersion 标准化 Windows 版本格式
// 将 "10" 转换为 "10.0"，"11" 转换为 "11.0"
func normalizeWindowsVersion(version string) string {
	if !strings.Contains(version, ".") {
		return version + ".0"
	}
	return version
}

// generateAcceptLanguage 根据指纹信息生成 Accept-Language 头
func (cli *Client) generateAcceptLanguage() string {
	fp := cli.getFingerprintInfo()
	
	lang := fp.LocaleLanguage
	country := fp.LocaleCountry
	
	// 如果没有语言信息，使用默认值
	if lang == "" {
		lang = "en"
	}
	if country == "" {
		country = "US"
	}
	
	// 根据语言和国家生成 Accept-Language
	// 格式：en-US,en;q=0.9 或 hi-IN,hi;q=0.9,en;q=0.8
	primary := fmt.Sprintf("%s-%s", lang, country)
	
	// 根据地区添加次要语言
	switch country {
	case "IN":
		// 印度：主要语言可能是 hi 或 en
		if lang == "hi" {
			return fmt.Sprintf("%s,hi;q=0.9,en-IN;q=0.8,en;q=0.7", primary)
		} else {
			return fmt.Sprintf("%s,en;q=0.9,hi-IN;q=0.8,hi;q=0.7", primary)
		}
	case "BR":
		// 巴西：主要语言是 pt
		return fmt.Sprintf("%s,pt;q=0.9,en-US;q=0.8,en;q=0.7", primary)
	default:
		// 默认：英语为主
		return fmt.Sprintf("%s,%s;q=0.9,en;q=0.8", primary, lang)
	}
}

// setBrowserHeaders 为 HTTP 请求设置完整的浏览器头
func (cli *Client) setBrowserHeaders(req *http.Request, isWebSocket bool) {
	userAgent := cli.generateUserAgent()
	req.Header.Set("User-Agent", userAgent)
	
	if isWebSocket {
		// WebSocket 连接头
		req.Header.Set("Origin", socket.Origin)
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "websocket")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
	} else {
		// HTTP 请求头（媒体上传/下载）
		req.Header.Set("Origin", socket.Origin)
		req.Header.Set("Referer", socket.Origin+"/")
		// Accept 头应该更具体，根据请求类型设置
		// 对于媒体上传/下载，使用通用的媒体类型
		req.Header.Set("Accept", "application/json, text/plain, */*")
		// 根据指纹信息生成 Accept-Language
		req.Header.Set("Accept-Language", cli.generateAcceptLanguage())
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		// 添加 Accept-Encoding（真实浏览器会发送）
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	}
}

// setWebSocketHeaders 为 WebSocket 连接设置浏览器头
func (cli *Client) setWebSocketHeaders(headers http.Header) {
	userAgent := cli.generateUserAgent()
	headers.Set("User-Agent", userAgent)
	headers.Set("Origin", socket.Origin)
	headers.Set("Sec-Fetch-Dest", "empty")
	headers.Set("Sec-Fetch-Mode", "websocket")
	headers.Set("Sec-Fetch-Site", "cross-site")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Pragma", "no-cache")
}
