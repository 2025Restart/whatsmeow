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
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/socket"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// generateUserAgent 根据 Payload 信息生成浏览器 User-Agent 字符串
// 优先从 ClientPayload 中提取，确保与握手包 100% 一致
func (cli *Client) generateUserAgent() string {
	payload := cli.Store.GetClientPayload()
	if payload == nil || payload.UserAgent == nil {
		// Fallback: 如果没有 Payload，尝试使用指纹
		fp := cli.getFingerprintInfo()
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
		return generateBrowserUserAgent(platformType, osName, osVersion)
	}

	// 核心：从 Payload 提取字段
	ua := payload.UserAgent
	osName := "Windows" // 默认
	if ua.GetManufacturer() == "Apple" {
		osName = "macOS"
	}
	
	// 从 PlatformType 推断
	platformType := waCompanionReg.DeviceProps_CHROME
	if payload.WebInfo != nil {
		// TODO: WebSubPlatform 映射到 DeviceProps_PlatformType
	}
	
	return generateBrowserUserAgent(platformType, osName, ua.GetOsVersion())
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
	// 根据地区添加次要语言
	switch country {
	case "IN":
		// 印度本地语种：不再包含 en
		return fmt.Sprintf("%s-IN,%s;q=0.9", lang, lang)
	case "BR":
		// 巴西：包含 pt-BR 和可能的 en-US
		if lang == "en" {
			return "en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7"
		}
		return "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7"
	default:
		// 默认：英语为主
		return fmt.Sprintf("%s-%s,%s;q=0.9,en-US;q=0.8,en;q=0.7", lang, country, lang)
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

	// 注入现代浏览器 Client Hints (解决 cln/cco/atn 报错)
	// 确保获取最新的 Payload（如果 GetClientPayload 已设置，使用它以确保一致性）
	var payload *waWa6.ClientPayload
	if cli.GetClientPayload != nil {
		payload = cli.GetClientPayload()
	} else {
		payload = cli.Store.GetClientPayload()
	}
	if payload != nil && payload.UserAgent != nil {
		osName := "Windows"
		if payload.UserAgent.GetManufacturer() == "Apple" {
			osName = "macOS"
		} else if payload.UserAgent.GetManufacturer() == "Microsoft" {
			osName = "Windows"
		}
		
		// 确保 Client Hints 格式正确（避免 atn/cln 封控）
		headers.Set("Sec-Ch-Ua-Platform", fmt.Sprintf("\"%s\"", osName))
		headers.Set("Sec-Ch-Ua-Mobile", "?0") // 强制锁定为 PC 端
		headers.Set("Sec-Ch-Ua", "\"Google Chrome\";v=\"131\", \"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\"")
		
		// 记录 Client Hints 设置（用于验证）
		if cli.Log != nil {
			cli.Log.Debugf("[Fingerprint] Set Client Hints: Platform=%s, Mobile=?0, UA=Chrome/131, MCC=%s, MNC=%s", 
				osName, payload.UserAgent.GetMcc(), payload.UserAgent.GetMnc())
			
			// 验证 WEB 平台特征一致性（避免 vll/lla/atn/cln 封控）
			if payload.UserAgent.GetPlatform() == waWa6.ClientPayload_UserAgent_WEB {
				issues := []string{}
				if payload.UserAgent.OsBuildNumber != nil {
					issues = append(issues, fmt.Sprintf("OsBuildNumber=%s (should be nil)", payload.UserAgent.GetOsBuildNumber()))
				}
				if payload.UserAgent.DeviceBoard != nil {
					issues = append(issues, fmt.Sprintf("DeviceBoard=%s (should be nil)", payload.UserAgent.GetDeviceBoard()))
				}
				if payload.UserAgent.DeviceModelType != nil {
					issues = append(issues, fmt.Sprintf("DeviceModelType=%s (should be nil)", payload.UserAgent.GetDeviceModelType()))
				}
				if payload.UserAgent.GetDevice() != "Desktop" {
					issues = append(issues, fmt.Sprintf("Device=%s (should be Desktop)", payload.UserAgent.GetDevice()))
				}
				if len(issues) > 0 {
					cli.Log.Warnf("[Fingerprint] WEB platform validation issues: %v", issues)
				}
			}
		}
		
		// 确保 WebInfo 存在（避免 lla 封控）
		if payload.WebInfo == nil {
			// WebInfo 应该在 BaseClientPayload 中已设置，但如果缺失则记录警告
			// 这里不设置，因为应该在 ApplyFingerprint 中处理
			if cli.Log != nil {
				cli.Log.Warnf("[Fingerprint] WebInfo is nil in payload, should be set by ApplyFingerprint")
			}
		} else if payload.WebInfo.WebSubPlatform == nil {
			if cli.Log != nil {
				cli.Log.Warnf("[Fingerprint] WebInfo.WebSubPlatform is nil, should be set by ApplyFingerprint")
			}
		}
	} else {
		// 如果 payload 为空，使用默认值
		headers.Set("Sec-Ch-Ua-Platform", "\"Windows\"")
		headers.Set("Sec-Ch-Ua-Mobile", "?0")
		headers.Set("Sec-Ch-Ua", "\"Google Chrome\";v=\"131\", \"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\"")
		if cli.Log != nil {
			cli.Log.Debugf("[Fingerprint] Set default Client Hints (payload is nil)")
		}
	}
}
