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
				OSVersions:   []string{"16", "17"},
			},
		},

		MobileNetworks: []fingerprint.MobileNetworkConfig{
			{MCC: "724", MNC: "02", OperatorName: "TIM", Weight: 0.30},
			{MCC: "724", MNC: "05", OperatorName: "Claro", Weight: 0.35},
			{MCC: "724", MNC: "10", OperatorName: "Vivo", Weight: 0.25},
			{MCC: "724", MNC: "11", OperatorName: "Oi", Weight: 0.10},
		},

		DeviceModels: map[string][]fingerprint.DeviceModelConfig{
			"Samsung": {
				{Model: "Galaxy A32", ModelType: "SM-A325F", Board: "samsung"},
				{Model: "Galaxy A52", ModelType: "SM-A525F", Board: "samsung"},
				{Model: "Galaxy M21", ModelType: "SM-M215F", Board: "samsung"},
			},
			"Motorola": {
				{Model: "Moto G60", ModelType: "XT2135-1", Board: "lahaina"},
				{Model: "Moto G50", ModelType: "XT2137-2", Board: "lahaina"},
				{Model: "Moto E20", ModelType: "XT2155-1", Board: "lahaina"},
			},
			"Xiaomi": {
				{Model: "Redmi Note 10", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "Redmi 9", ModelType: "M2004J19G", Board: "lito"},
			},
			"Apple": {
				{Model: "iPhone 11", ModelType: "iPhone12,1", Board: "iPhone"},
				{Model: "iPhone 12", ModelType: "iPhone13,2", Board: "iPhone"},
				{Model: "iPhone XR", ModelType: "iPhone11,8", Board: "iPhone"},
			},
		},
	}

	_ = fingerprint.RegisterRegion(config)
}
