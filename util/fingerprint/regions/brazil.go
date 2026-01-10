// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package regions

import (
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/util/fingerprint"
)

func init() {
	config := &fingerprint.RegionConfig{
		Code: "BR",
		Name: "Brazil",

		Languages: []fingerprint.LanguageConfig{
			{Code: "pt", Weight: 1.0, Countries: []string{"BR"}}, // 葡萄牙语 100%
		},

		DeviceDistribution: []fingerprint.PlatformDistribution{
			{
				Platform:     waWa6.ClientPayload_UserAgent_ANDROID,
				PlatformType: waCompanionReg.DeviceProps_ANDROID_PHONE,
				DeviceType:   waWa6.ClientPayload_UserAgent_PHONE,
				Weight:       0.80, // 80% Android
				OSName:       "Android",
				OSVersions:   []string{"11", "12", "13"},
			},
			{
				Platform:     waWa6.ClientPayload_UserAgent_IOS,
				PlatformType: waCompanionReg.DeviceProps_IOS_PHONE,
				DeviceType:   waWa6.ClientPayload_UserAgent_PHONE,
				Weight:       0.20, // 20% iOS
				OSName:       "iOS",
				OSVersions:   []string{"15", "16", "17"},
			},
		},

		MobileNetworks: []fingerprint.MobileNetworkConfig{
			{MCC: "724", MNC: "02", OperatorName: "TIM", Weight: 0.30},
			{MCC: "724", MNC: "05", OperatorName: "Claro", Weight: 0.35},
			{MCC: "724", MNC: "10", OperatorName: "Vivo", Weight: 0.25},
			{MCC: "724", MNC: "11", OperatorName: "Oi", Weight: 0.10},
		},

		DeviceModels: map[string][]fingerprint.DeviceModelConfig{
			"Motorola": {
				// Moto G 系列（巴西市场主流，25+ 型号）
				{Model: "Moto G60", ModelType: "XT2135-1", Board: "lahaina"},
				{Model: "Moto G60S", ModelType: "XT2133-1", Board: "lahaina"},
				{Model: "Moto G50", ModelType: "XT2137-2", Board: "lahaina"},
				{Model: "Moto G52", ModelType: "XT2221-1", Board: "lahaina"},
				{Model: "Moto G62", ModelType: "XT2223-1", Board: "lahaina"},
				{Model: "Moto G71", ModelType: "XT2169-1", Board: "lahaina"},
				{Model: "Moto G82", ModelType: "XT2225-1", Board: "lahaina"},
				{Model: "Moto G100", ModelType: "XT2125-1", Board: "lahaina"},
				{Model: "Moto G200", ModelType: "XT2175-1", Board: "lahaina"},
				{Model: "Moto G30", ModelType: "XT2129-1", Board: "lahaina"},
				{Model: "Moto G31", ModelType: "XT2173-1", Board: "lahaina"},
				{Model: "Moto G32", ModelType: "XT2235-1", Board: "lahaina"},
				{Model: "Moto G42", ModelType: "XT2233-1", Board: "lahaina"},
				{Model: "Moto G72", ModelType: "XT2255-1", Board: "lahaina"},
				{Model: "Moto G73", ModelType: "XT2237-1", Board: "lahaina"},
				// Moto E 系列
				{Model: "Moto E20", ModelType: "XT2155-1", Board: "lahaina"},
				{Model: "Moto E22", ModelType: "XT2239-1", Board: "lahaina"},
				{Model: "Moto E30", ModelType: "XT2157-1", Board: "lahaina"},
				{Model: "Moto E32", ModelType: "XT2235-1", Board: "lahaina"},
				// Moto Edge 系列
				{Model: "Moto Edge 20", ModelType: "XT2143-1", Board: "lahaina"},
				{Model: "Moto Edge 30", ModelType: "XT2203-1", Board: "lahaina"},
				{Model: "Moto Edge 40", ModelType: "XT2301-1", Board: "lahaina"},
			},
			"Samsung": {
				// Galaxy A 系列（巴西市场，20+ 型号）
				{Model: "Galaxy A32", ModelType: "SM-A325F", Board: "samsung"},
				{Model: "Galaxy A32 5G", ModelType: "SM-A326B", Board: "samsung"},
				{Model: "Galaxy A52", ModelType: "SM-A525F", Board: "samsung"},
				{Model: "Galaxy A52s", ModelType: "SM-A528B", Board: "samsung"},
				{Model: "Galaxy A53", ModelType: "SM-A536B", Board: "samsung"},
				{Model: "Galaxy A54", ModelType: "SM-A546B", Board: "samsung"},
				{Model: "Galaxy A13", ModelType: "SM-A135F", Board: "samsung"},
				{Model: "Galaxy A14", ModelType: "SM-A145F", Board: "samsung"},
				{Model: "Galaxy A23", ModelType: "SM-A235F", Board: "samsung"},
				{Model: "Galaxy A33", ModelType: "SM-A336B", Board: "samsung"},
				{Model: "Galaxy A73", ModelType: "SM-A736B", Board: "samsung"},
				{Model: "Galaxy A34", ModelType: "SM-A346B", Board: "samsung"},
				// Galaxy M 系列
				{Model: "Galaxy M21", ModelType: "SM-M215F", Board: "samsung"},
				{Model: "Galaxy M32", ModelType: "SM-M325F", Board: "samsung"},
				{Model: "Galaxy M33", ModelType: "SM-M336B", Board: "samsung"},
				{Model: "Galaxy M52", ModelType: "SM-M526B", Board: "samsung"},
				{Model: "Galaxy M53", ModelType: "SM-M536B", Board: "samsung"},
			},
			"Xiaomi": {
				// Redmi 系列（巴西市场，15+ 型号）
				{Model: "Redmi Note 10", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "Redmi Note 10 Pro", ModelType: "M2101K6G", Board: "lito"},
				{Model: "Redmi Note 11", ModelType: "M2201K7AG", Board: "lito"},
				{Model: "Redmi Note 11 Pro", ModelType: "M2201K6G", Board: "lito"},
				{Model: "Redmi Note 12", ModelType: "M2211K6G", Board: "lito"},
				{Model: "Redmi 9", ModelType: "M2004J19G", Board: "lito"},
				{Model: "Redmi 9A", ModelType: "M2006C3LG", Board: "lito"},
				{Model: "Redmi 10", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "Redmi 10A", ModelType: "M2201K7AG", Board: "lito"},
				{Model: "Redmi 11", ModelType: "M2211K6G", Board: "lito"},
				// POCO 系列
				{Model: "POCO M3", ModelType: "M2010J19CG", Board: "lito"},
				{Model: "POCO M4", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "POCO X3", ModelType: "M2007J20CG", Board: "lito"},
				{Model: "POCO X4", ModelType: "M2201K6G", Board: "lito"},
			},
			"LG": {
				// LG 手机（巴西市场，10+ 型号）
				{Model: "LG K40", ModelType: "LM-X420", Board: "lahaina"},
				{Model: "LG K50", ModelType: "LM-X520", Board: "lahaina"},
				{Model: "LG K61", ModelType: "LM-Q630", Board: "lahaina"},
				{Model: "LG Velvet", ModelType: "LM-G900", Board: "lahaina"},
				{Model: "LG G8", ModelType: "LM-G820", Board: "lahaina"},
				{Model: "LG V60", ModelType: "LM-V600", Board: "lahaina"},
			},
			"Apple": {
				{Model: "iPhone XR", ModelType: "iPhone11,8", Board: "iPhone"},
				{Model: "iPhone 11", ModelType: "iPhone12,1", Board: "iPhone"},
				{Model: "iPhone 12", ModelType: "iPhone13,2", Board: "iPhone"},
				{Model: "iPhone 12 mini", ModelType: "iPhone13,1", Board: "iPhone"},
				{Model: "iPhone 13", ModelType: "iPhone14,2", Board: "iPhone"},
				{Model: "iPhone 13 mini", ModelType: "iPhone14,1", Board: "iPhone"},
				{Model: "iPhone 14", ModelType: "iPhone14,7", Board: "iPhone"},
				{Model: "iPhone 14 Plus", ModelType: "iPhone14,8", Board: "iPhone"},
				{Model: "iPhone 15", ModelType: "iPhone15,2", Board: "iPhone"},
				{Model: "iPhone 15 Plus", ModelType: "iPhone15,3", Board: "iPhone"},
				{Model: "iPhone SE 2022", ModelType: "iPhone14,6", Board: "iPhone"},
			},
		},
	}

	_ = fingerprint.RegisterRegion(config)
}
