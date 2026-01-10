// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package regions

import (
	"go.mau.fi/whatsmeow/util/fingerprint"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
)

func init() {
	// 默认配置（向后兼容，用于未指定地区的情况）
	config := &fingerprint.RegionConfig{
		Code: "",
		Name: "Default",

		Languages: []fingerprint.LanguageConfig{
			{Code: "en", Weight: 1.0, Countries: []string{"US", "GB", "CA", "AU", "NZ"}},
		},

		DeviceDistribution: []fingerprint.PlatformDistribution{
			{
				Platform:     waWa6.ClientPayload_UserAgent_MACOS,
				PlatformType: waCompanionReg.DeviceProps_CATALINA,
				DeviceType:   waWa6.ClientPayload_UserAgent_DESKTOP,
				Weight:       1.0,
				OSName:       "macOS",
				OSVersions:   []string{"10.15.7", "11.7.10", "12.6.1", "13.5.2", "14.2.1"},
			},
		},

		MobileNetworks: []fingerprint.MobileNetworkConfig{
			{MCC: "310", MNC: "260", OperatorName: "T-Mobile", Weight: 1.0},
		},

		DeviceModels: map[string][]fingerprint.DeviceModelConfig{
			"Apple": {
				{Model: "MacBook Pro", ModelType: "MacBookPro18,1", Board: "Mac-"},
				{Model: "MacBook Air", ModelType: "MacBookAir10,1", Board: "Mac-"},
			},
			"Microsoft": {
				{Model: "Surface Pro", ModelType: "SurfacePro9", Board: "Surface"},
			},
		},
	}

	// 设置为默认配置
	_ = fingerprint.SetDefaultRegion(config)
}
