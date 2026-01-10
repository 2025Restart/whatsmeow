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
	config := &fingerprint.RegionConfig{
		Code: "IN",
		Name: "India",

		Languages: []fingerprint.LanguageConfig{
			{Code: "hi", Weight: 0.4, Countries: []string{"IN"}}, // 印地语 40%
			{Code: "en", Weight: 0.6, Countries: []string{"IN"}}, // 英语 60%
		},

		DeviceDistribution: []fingerprint.PlatformDistribution{
			{
				Platform:     waWa6.ClientPayload_UserAgent_ANDROID,
				PlatformType: waCompanionReg.DeviceProps_ANDROID_PHONE,
				DeviceType:   waWa6.ClientPayload_UserAgent_PHONE,
				Weight:       0.75, // 75% Android
				OSName:       "Android",
				OSVersions:   []string{"11", "12", "13", "14"},
			},
			{
				Platform:     waWa6.ClientPayload_UserAgent_IOS,
				PlatformType: waCompanionReg.DeviceProps_IOS_PHONE,
				DeviceType:   waWa6.ClientPayload_UserAgent_PHONE,
				Weight:       0.25, // 25% iOS
				OSName:       "iOS",
				OSVersions:   []string{"16", "17"},
			},
		},

		MobileNetworks: []fingerprint.MobileNetworkConfig{
			{MCC: "404", MNC: "01", OperatorName: "Airtel", Weight: 0.30},
			{MCC: "404", MNC: "07", OperatorName: "Jio", Weight: 0.35},
			{MCC: "404", MNC: "20", OperatorName: "Vodafone", Weight: 0.20},
			{MCC: "404", MNC: "11", OperatorName: "BSNL", Weight: 0.15},
		},

		DeviceModels: map[string][]fingerprint.DeviceModelConfig{
			"Samsung": {
				{Model: "Galaxy A52", ModelType: "SM-A525F", Board: "samsung"},
				{Model: "Galaxy M31", ModelType: "SM-M315F", Board: "samsung"},
				{Model: "Galaxy A32", ModelType: "SM-A325F", Board: "samsung"},
			},
			"Xiaomi": {
				{Model: "Redmi Note 10", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "Redmi 9", ModelType: "M2004J19G", Board: "lito"},
				{Model: "Redmi Note 11", ModelType: "M2201K7AG", Board: "lito"},
			},
			"OnePlus": {
				{Model: "Nord CE", ModelType: "EB2101", Board: "lahaina"},
				{Model: "9R", ModelType: "LE2101", Board: "lahaina"},
			},
			"Apple": {
				{Model: "iPhone 12", ModelType: "iPhone13,2", Board: "iPhone"},
				{Model: "iPhone 13", ModelType: "iPhone14,2", Board: "iPhone"},
				{Model: "iPhone 11", ModelType: "iPhone12,1", Board: "iPhone"},
			},
		},
	}

	_ = fingerprint.RegisterRegion(config)
}
