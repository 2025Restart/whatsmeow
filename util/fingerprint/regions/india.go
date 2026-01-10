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
				OSVersions:   []string{"15", "16", "17"},
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
				// Galaxy A 系列（印度市场主流，30+ 型号）
				{Model: "Galaxy A52", ModelType: "SM-A525F", Board: "samsung"},
				{Model: "Galaxy A52s", ModelType: "SM-A528B", Board: "samsung"},
				{Model: "Galaxy A32", ModelType: "SM-A325F", Board: "samsung"},
				{Model: "Galaxy A32 5G", ModelType: "SM-A326B", Board: "samsung"},
				{Model: "Galaxy A22", ModelType: "SM-A225F", Board: "samsung"},
				{Model: "Galaxy A22 5G", ModelType: "SM-A226B", Board: "samsung"},
				{Model: "Galaxy A12", ModelType: "SM-A125F", Board: "samsung"},
				{Model: "Galaxy A13", ModelType: "SM-A135F", Board: "samsung"},
				{Model: "Galaxy A14", ModelType: "SM-A145F", Board: "samsung"},
				{Model: "Galaxy A23", ModelType: "SM-A235F", Board: "samsung"},
				{Model: "Galaxy A33", ModelType: "SM-A336B", Board: "samsung"},
				{Model: "Galaxy A53", ModelType: "SM-A536B", Board: "samsung"},
				{Model: "Galaxy A54", ModelType: "SM-A546B", Board: "samsung"},
				{Model: "Galaxy A73", ModelType: "SM-A736B", Board: "samsung"},
				{Model: "Galaxy A34", ModelType: "SM-A346B", Board: "samsung"},
				// Galaxy M 系列（印度市场专属，15+ 型号）
				{Model: "Galaxy M31", ModelType: "SM-M315F", Board: "samsung"},
				{Model: "Galaxy M32", ModelType: "SM-M325F", Board: "samsung"},
				{Model: "Galaxy M33", ModelType: "SM-M336B", Board: "samsung"},
				{Model: "Galaxy M42", ModelType: "SM-M426B", Board: "samsung"},
				{Model: "Galaxy M52", ModelType: "SM-M526B", Board: "samsung"},
				{Model: "Galaxy M53", ModelType: "SM-M536B", Board: "samsung"},
				{Model: "Galaxy M21", ModelType: "SM-M215F", Board: "samsung"},
				{Model: "Galaxy M22", ModelType: "SM-M225FV", Board: "samsung"},
				{Model: "Galaxy M12", ModelType: "SM-M127F", Board: "samsung"},
				{Model: "Galaxy M13", ModelType: "SM-M135F", Board: "samsung"},
				{Model: "Galaxy M14", ModelType: "SM-M146B", Board: "samsung"},
				// Galaxy F 系列
				{Model: "Galaxy F23", ModelType: "SM-E236B", Board: "samsung"},
				{Model: "Galaxy F13", ModelType: "SM-E135F", Board: "samsung"},
				{Model: "Galaxy F14", ModelType: "SM-E146B", Board: "samsung"},
			},
			"Xiaomi": {
				// Redmi 系列（印度市场主流，20+ 型号）
				{Model: "Redmi Note 10", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "Redmi Note 10 Pro", ModelType: "M2101K6G", Board: "lito"},
				{Model: "Redmi Note 11", ModelType: "M2201K7AG", Board: "lito"},
				{Model: "Redmi Note 11 Pro", ModelType: "M2201K6G", Board: "lito"},
				{Model: "Redmi Note 12", ModelType: "M2211K6G", Board: "lito"},
				{Model: "Redmi Note 12 Pro", ModelType: "M2211K6G", Board: "lito"},
				{Model: "Redmi Note 13", ModelType: "M2311K6G", Board: "lito"},
				{Model: "Redmi 9", ModelType: "M2004J19G", Board: "lito"},
				{Model: "Redmi 9A", ModelType: "M2006C3LG", Board: "lito"},
				{Model: "Redmi 10", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "Redmi 10A", ModelType: "M2201K7AG", Board: "lito"},
				{Model: "Redmi 11", ModelType: "M2211K6G", Board: "lito"},
				{Model: "Redmi 12", ModelType: "M2311K6G", Board: "lito"},
				// POCO 系列
				{Model: "POCO M3", ModelType: "M2010J19CG", Board: "lito"},
				{Model: "POCO M4", ModelType: "M2101K7AG", Board: "lito"},
				{Model: "POCO X3", ModelType: "M2007J20CG", Board: "lito"},
				{Model: "POCO X4", ModelType: "M2201K6G", Board: "lito"},
				{Model: "POCO F3", ModelType: "M2012K11AG", Board: "lito"},
				{Model: "POCO F4", ModelType: "M2201K6G", Board: "lito"},
			},
			"OnePlus": {
				{Model: "Nord CE", ModelType: "EB2101", Board: "lahaina"},
				{Model: "Nord CE 2", ModelType: "EB2103", Board: "lahaina"},
				{Model: "Nord 2", ModelType: "DN2101", Board: "lahaina"},
				{Model: "9R", ModelType: "LE2101", Board: "lahaina"},
				{Model: "9", ModelType: "LE2111", Board: "lahaina"},
				{Model: "10R", ModelType: "CPH2411", Board: "lahaina"},
				{Model: "10 Pro", ModelType: "NE2211", Board: "lahaina"},
				{Model: "11R", ModelType: "CPH2487", Board: "lahaina"},
				{Model: "11", ModelType: "CPH2449", Board: "lahaina"},
				{Model: "Nord CE 3", ModelType: "CPH2569", Board: "lahaina"},
			},
			"Realme": {
				{Model: "Realme 8", ModelType: "RMX3085", Board: "lahaina"},
				{Model: "Realme 8 Pro", ModelType: "RMX3081", Board: "lahaina"},
				{Model: "Realme 9", ModelType: "RMX3521", Board: "lahaina"},
				{Model: "Realme 9 Pro", ModelType: "RMX3471", Board: "lahaina"},
				{Model: "Realme 10", ModelType: "RMX3630", Board: "lahaina"},
				{Model: "Realme 10 Pro", ModelType: "RMX3661", Board: "lahaina"},
				{Model: "Realme Narzo 50", ModelType: "RMX3286", Board: "lahaina"},
				{Model: "Realme Narzo 60", ModelType: "RMX3751", Board: "lahaina"},
				{Model: "Realme C25", ModelType: "RMX3195", Board: "lahaina"},
				{Model: "Realme C35", ModelType: "RMX3511", Board: "lahaina"},
			},
			"Vivo": {
				{Model: "Vivo Y20", ModelType: "V2027", Board: "lahaina"},
				{Model: "Vivo Y21", ModelType: "V2111", Board: "lahaina"},
				{Model: "Vivo Y33s", ModelType: "V2115", Board: "lahaina"},
				{Model: "Vivo Y55", ModelType: "V2204", Board: "lahaina"},
				{Model: "Vivo T1", ModelType: "V2111", Board: "lahaina"},
				{Model: "Vivo T1 Pro", ModelType: "V2115", Board: "lahaina"},
				{Model: "Vivo V23", ModelType: "V2115", Board: "lahaina"},
				{Model: "Vivo V25", ModelType: "V2204", Board: "lahaina"},
			},
			"Oppo": {
				{Model: "Oppo A54", ModelType: "CPH2239", Board: "lahaina"},
				{Model: "Oppo A74", ModelType: "CPH2219", Board: "lahaina"},
				{Model: "Oppo A96", ModelType: "CPH2365", Board: "lahaina"},
				{Model: "Oppo F19", ModelType: "CPH2219", Board: "lahaina"},
				{Model: "Oppo F21", ModelType: "CPH2451", Board: "lahaina"},
				{Model: "Oppo Reno 7", ModelType: "CPH2363", Board: "lahaina"},
				{Model: "Oppo Reno 8", ModelType: "CPH2451", Board: "lahaina"},
			},
			"Apple": {
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
