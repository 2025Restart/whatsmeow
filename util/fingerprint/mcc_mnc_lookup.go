// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

// MCCMNCEntry MCC/MNC 查找表条目
type MCCMNCEntry struct {
	MCC          string
	MNC          string
	CountryCode  string
	OperatorName string
	IsCommon     bool // 是否为该 MCC 的常见 MNC
}

// mccMNCList MCC/MNC 查找表（按 MCC 组织）
var mccMNCList = map[string][]MCCMNCEntry{
	// 印度 (IN)
	"404": {
		{MCC: "404", MNC: "01", CountryCode: "IN", OperatorName: "Vi", IsCommon: true},
		{MCC: "404", MNC: "02", CountryCode: "IN", OperatorName: "Airtel", IsCommon: true},
		{MCC: "404", MNC: "03", CountryCode: "IN", OperatorName: "Airtel", IsCommon: true},
		{MCC: "404", MNC: "04", CountryCode: "IN", OperatorName: "Vi", IsCommon: false},
		{MCC: "404", MNC: "05", CountryCode: "IN", OperatorName: "Vi", IsCommon: false},
		{MCC: "404", MNC: "07", CountryCode: "IN", OperatorName: "Vi", IsCommon: false},
		{MCC: "404", MNC: "10", CountryCode: "IN", OperatorName: "Airtel", IsCommon: false},
		{MCC: "404", MNC: "11", CountryCode: "IN", OperatorName: "BSNL", IsCommon: false},
		{MCC: "404", MNC: "15", CountryCode: "IN", OperatorName: "BSNL", IsCommon: false},
		{MCC: "404", MNC: "20", CountryCode: "IN", OperatorName: "Vodafone Idea", IsCommon: true},
		{MCC: "404", MNC: "22", CountryCode: "IN", OperatorName: "Vodafone Idea", IsCommon: false},
		{MCC: "404", MNC: "27", CountryCode: "IN", OperatorName: "Vodafone Idea", IsCommon: false},
		{MCC: "404", MNC: "34", CountryCode: "IN", OperatorName: "BSNL", IsCommon: false},
		{MCC: "404", MNC: "38", CountryCode: "IN", OperatorName: "BSNL", IsCommon: false},
		{MCC: "404", MNC: "45", CountryCode: "IN", OperatorName: "Airtel", IsCommon: false},
	},
	"405": {
		// Jio 的有效 MNC 范围：840, 854-874
		{MCC: "405", MNC: "840", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "854", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "855", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "856", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "857", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "858", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "859", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "860", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "861", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "862", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "863", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "864", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "865", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "866", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "867", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "868", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "869", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "870", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "871", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "872", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "873", CountryCode: "IN", OperatorName: "Jio", IsCommon: false},
		{MCC: "405", MNC: "874", CountryCode: "IN", OperatorName: "Jio", IsCommon: true},
	},
	// 巴西 (BR)
	"724": {
		{MCC: "724", MNC: "02", CountryCode: "BR", OperatorName: "TIM", IsCommon: true},
		{MCC: "724", MNC: "03", CountryCode: "BR", OperatorName: "TIM", IsCommon: false},
		{MCC: "724", MNC: "04", CountryCode: "BR", OperatorName: "Claro", IsCommon: false},
		{MCC: "724", MNC: "05", CountryCode: "BR", OperatorName: "Claro", IsCommon: true},
		{MCC: "724", MNC: "06", CountryCode: "BR", OperatorName: "Vivo", IsCommon: false},
		{MCC: "724", MNC: "10", CountryCode: "BR", OperatorName: "Vivo", IsCommon: true},
		{MCC: "724", MNC: "11", CountryCode: "BR", OperatorName: "Vivo", IsCommon: false},
		{MCC: "724", MNC: "14", CountryCode: "BR", OperatorName: "Brasil Telecom", IsCommon: false},
		{MCC: "724", MNC: "15", CountryCode: "BR", OperatorName: "Brasil Telecom", IsCommon: false},
		{MCC: "724", MNC: "16", CountryCode: "BR", OperatorName: "Brasil Telecom GSM", IsCommon: false},
		{MCC: "724", MNC: "23", CountryCode: "BR", OperatorName: "Vivo", IsCommon: false},
		{MCC: "724", MNC: "30", CountryCode: "BR", OperatorName: "Oi", IsCommon: false},
		{MCC: "724", MNC: "31", CountryCode: "BR", OperatorName: "Oi", IsCommon: true},
		{MCC: "724", MNC: "32", CountryCode: "BR", OperatorName: "Claro", IsCommon: false},
		{MCC: "724", MNC: "33", CountryCode: "BR", OperatorName: "Claro", IsCommon: false},
		{MCC: "724", MNC: "34", CountryCode: "BR", OperatorName: "Claro", IsCommon: false},
		{MCC: "724", MNC: "38", CountryCode: "BR", OperatorName: "Claro", IsCommon: false},
	},
}

// MNCToMCCMap MNC 到 MCC 的映射（需要结合国家代码）
// 格式: "countryCode:mnc" -> mcc
var mncToMCCMap = map[string]string{
	// 印度
	"IN:01": "404", "IN:02": "404", "IN:03": "404", "IN:04": "404", "IN:05": "404",
	"IN:07": "404", "IN:10": "404", "IN:11": "404", "IN:15": "404", "IN:20": "404",
	"IN:22": "404", "IN:27": "404", "IN:34": "404", "IN:38": "404", "IN:45": "404",
	"IN:840": "405", "IN:854": "405", "IN:855": "405", "IN:856": "405", "IN:857": "405",
	"IN:858": "405", "IN:859": "405", "IN:860": "405", "IN:861": "405", "IN:862": "405",
	"IN:863": "405", "IN:864": "405", "IN:865": "405", "IN:866": "405", "IN:867": "405",
	"IN:868": "405", "IN:869": "405", "IN:870": "405", "IN:871": "405", "IN:872": "405",
	"IN:873": "405", "IN:874": "405",
	// 巴西
	"BR:02": "724", "BR:03": "724", "BR:04": "724", "BR:05": "724", "BR:06": "724",
	"BR:10": "724", "BR:11": "724", "BR:14": "724", "BR:15": "724", "BR:16": "724",
	"BR:30": "724", "BR:31": "724", "BR:32": "724", "BR:33": "724", "BR:34": "724", "BR:38": "724",
}

// InferMNCFromMCC 根据 MCC 推断 MNC（返回最常见的 MNC）
func InferMNCFromMCC(mcc string) string {
	if entries, ok := mccMNCList[mcc]; ok && len(entries) > 0 {
		// 优先返回标记为常见的 MNC
		for _, entry := range entries {
			if entry.IsCommon {
				return entry.MNC
			}
		}
		// 如果没有常见的，返回第一个
		return entries[0].MNC
	}
	return ""
}

// InferMCCFromMNC 根据 MNC 和国家代码推断 MCC
func InferMCCFromMNC(mnc, countryCode string) string {
	if countryCode == "" {
		return ""
	}
	key := countryCode + ":" + mnc
	if mcc, ok := mncToMCCMap[key]; ok {
		return mcc
	}
	return ""
}

// ValidateMCCMNC 验证 MCC/MNC 组合是否有效
func ValidateMCCMNC(mcc, mnc string) bool {
	if entries, ok := mccMNCList[mcc]; ok {
		for _, entry := range entries {
			if entry.MNC == mnc {
				return true
			}
		}
	}
	return false
}

// GetCommonMNCForMCC 获取指定 MCC 的常见 MNC 列表
func GetCommonMNCForMCC(mcc string) []string {
	var result []string
	if entries, ok := mccMNCList[mcc]; ok {
		for _, entry := range entries {
			if entry.IsCommon {
				result = append(result, entry.MNC)
			}
		}
	}
	return result
}
