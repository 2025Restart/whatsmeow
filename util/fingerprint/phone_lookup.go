// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import (
	"strings"
)

// PhoneRegionInfo 手机号对应的地区信息
type PhoneRegionInfo struct {
	RegionCode   string
	MCC          string
	MNC          string
	OperatorName string
}

// LookupPhoneRegion 根据手机号段精确查找地区和运营商信息
func LookupPhoneRegion(phone string) *PhoneRegionInfo {
	if phone == "" {
		return nil
	}

	// 清理手机号
	cleanPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// 1. 印度 (91)
	if strings.HasPrefix(cleanPhone, "91") {
		return lookupIndia(cleanPhone[2:])
	}

	// 2. 巴西 (55)
	if strings.HasPrefix(cleanPhone, "55") {
		return lookupBrazil(cleanPhone[2:])
	}

	return nil
}

// lookupIndia 印度号段查找 (10位)
func lookupIndia(num string) *PhoneRegionInfo {
	if len(num) < 4 {
		return &PhoneRegionInfo{RegionCode: "IN", MCC: "404", MNC: "01", OperatorName: "Airtel"}
	}

	prefix4 := num[:4]
	
	// 简单映射逻辑（基于常用号段）
	// Jio (MCC 405 主要，或 404-09)
	if strings.HasPrefix(num, "70") || strings.HasPrefix(num, "79") || strings.HasPrefix(num, "6") || strings.HasPrefix(num, "8") {
		return &PhoneRegionInfo{RegionCode: "IN", MCC: "405", MNC: "865", OperatorName: "Jio"}
	}

	// Airtel (404-02, 03, 10...)
	airtelPrefixes := []string{"981", "987", "991", "997", "971"}
	for _, p := range airtelPrefixes {
		if strings.HasPrefix(num, p) {
			return &PhoneRegionInfo{RegionCode: "IN", MCC: "404", MNC: "02", OperatorName: "Airtel"}
		}
	}

	// Vi (404-01, 04, 05...)
	viPrefixes := []string{"982", "989", "992", "999", "972"}
	for _, p := range viPrefixes {
		if strings.HasPrefix(num, p) {
			return &PhoneRegionInfo{RegionCode: "IN", MCC: "404", MNC: "01", OperatorName: "Vi"}
		}
	}

	// BSNL (404-34, 38...)
	if strings.HasPrefix(num, "94") {
		return &PhoneRegionInfo{RegionCode: "IN", MCC: "404", MNC: "34", OperatorName: "BSNL"}
	}

	// 默认返回印度通用
	_ = prefix4
	return &PhoneRegionInfo{RegionCode: "IN", MCC: "404", MNC: "01", OperatorName: "Airtel"}
}

// lookupBrazil 巴西号段查找 (11位: AA 9NNNN-NNNN)
func lookupBrazil(num string) *PhoneRegionInfo {
	if len(num) < 4 {
		return &PhoneRegionInfo{RegionCode: "BR", MCC: "724", MNC: "02", OperatorName: "TIM"}
	}

	// 巴西号段逻辑：AA (2位区号) + 9 (移动标识) + N (号段)
	// ddd := num[:2]
	series := num[2:4]

	// Vivo (724-10, 11)
	if series >= "96" && series <= "99" {
		return &PhoneRegionInfo{RegionCode: "BR", MCC: "724", MNC: "10", OperatorName: "Vivo"}
	}

	// Claro (724-04, 05)
	if series >= "91" && series <= "94" {
		return &PhoneRegionInfo{RegionCode: "BR", MCC: "724", MNC: "05", OperatorName: "Claro"}
	}

	// TIM (724-02, 03)
	if series >= "81" && series <= "87" {
		return &PhoneRegionInfo{RegionCode: "BR", MCC: "724", MNC: "02", OperatorName: "TIM"}
	}

	// Oi (724-14, 15)
	if series == "80" || series == "88" || series == "89" {
		return &PhoneRegionInfo{RegionCode: "BR", MCC: "724", MNC: "16", OperatorName: "Oi"}
	}

	// 默认返回巴西通用
	return &PhoneRegionInfo{RegionCode: "BR", MCC: "724", MNC: "02", OperatorName: "TIM"}
}
