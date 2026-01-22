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
		Code: "IN",
		Name: "India",

		Languages: []fingerprint.LanguageConfig{
			{Code: "hi", Weight: 0.35, Countries: []string{"IN"}}, // 印地语 35%
			{Code: "ta", Weight: 0.15, Countries: []string{"IN"}}, // 泰米尔语 15%
			{Code: "te", Weight: 0.15, Countries: []string{"IN"}}, // 泰卢固语 15%
			{Code: "mr", Weight: 0.12, Countries: []string{"IN"}}, // 马拉地语 12%
			{Code: "bn", Weight: 0.13, Countries: []string{"IN"}}, // 孟加拉语 13%
			{Code: "kn", Weight: 0.10, Countries: []string{"IN"}}, // 卡纳达语 10%
		},

		DeviceDistribution: []fingerprint.PlatformDistribution{
			{
				Platform:     waWa6.ClientPayload_UserAgent_WEB,
				PlatformType: waCompanionReg.DeviceProps_CHROME,
				DeviceType:   waWa6.ClientPayload_UserAgent_DESKTOP,
				Weight:       0.70, // 70% Windows Web
				OSName:       "Windows",
				OSVersions:   []string{"10", "11"},
			},
			{
				Platform:     waWa6.ClientPayload_UserAgent_WEB,
				PlatformType: waCompanionReg.DeviceProps_CATALINA,
				DeviceType:   waWa6.ClientPayload_UserAgent_DESKTOP,
				Weight:       0.20, // 20% macOS
				OSName:       "macOS",
				OSVersions:   []string{"12", "13", "14"},
			},
			{
				Platform:     waWa6.ClientPayload_UserAgent_WEB,
				PlatformType: waCompanionReg.DeviceProps_DESKTOP,
				DeviceType:   waWa6.ClientPayload_UserAgent_DESKTOP,
				Weight:       0.10, // 10% Linux
				OSName:       "Linux",
				OSVersions:   []string{"5.15", "5.19", "6.1"},
			},
		},

		MobileNetworks: []fingerprint.MobileNetworkConfig{
			// MCC 404 (主要GSM网络)
			{MCC: "404", MNC: "01", OperatorName: "Airtel", Weight: 0.20},
			{MCC: "404", MNC: "07", OperatorName: "Jio", Weight: 0.25},
			{MCC: "404", MNC: "20", OperatorName: "Vodafone Idea", Weight: 0.15},
			{MCC: "404", MNC: "11", OperatorName: "BSNL", Weight: 0.10},
			{MCC: "404", MNC: "10", OperatorName: "Airtel", Weight: 0.04},
			{MCC: "404", MNC: "22", OperatorName: "Vodafone Idea", Weight: 0.03},
			{MCC: "404", MNC: "27", OperatorName: "Vodafone Idea", Weight: 0.02},
			{MCC: "404", MNC: "15", OperatorName: "BSNL", Weight: 0.02},
			{MCC: "404", MNC: "45", OperatorName: "Airtel", Weight: 0.01},
			// MCC 405 (CDMA/LTE网络) - 启用 Jio 的 LTE 网络配置
			{MCC: "405", MNC: "799", OperatorName: "Jio", Weight: 0.10},
			{MCC: "405", MNC: "800", OperatorName: "Jio", Weight: 0.02},
			{MCC: "405", MNC: "801", OperatorName: "Jio", Weight: 0.01},
		},

		DeviceModels: map[string][]fingerprint.DeviceModelConfig{
			"Apple": {
				// MacBook Pro 系列（印度市场主流，M1/M2/M3芯片）
				{Model: "MacBook Pro 14 M1", ModelType: "MacBookPro18,1", Board: "Mac-"},
				{Model: "MacBook Pro 16 M1", ModelType: "MacBookPro18,2", Board: "Mac-"},
				{Model: "MacBook Pro 14 M2", ModelType: "MacBookPro19,1", Board: "Mac-"},
				{Model: "MacBook Pro 16 M2", ModelType: "MacBookPro19,2", Board: "Mac-"},
				{Model: "MacBook Pro 14 M3", ModelType: "MacBookPro20,1", Board: "Mac-"},
				{Model: "MacBook Pro 16 M3", ModelType: "MacBookPro20,2", Board: "Mac-"},
				{Model: "MacBook Pro 13 M1", ModelType: "MacBookPro17,1", Board: "Mac-"},
				{Model: "MacBook Pro 13 M2", ModelType: "MacBookPro18,3", Board: "Mac-"},
				// MacBook Air 系列
				{Model: "MacBook Air M1", ModelType: "MacBookAir10,1", Board: "Mac-"},
				{Model: "MacBook Air M2", ModelType: "MacBookAir14,1", Board: "Mac-"},
				{Model: "MacBook Air M2 15", ModelType: "MacBookAir15,2", Board: "Mac-"},
				{Model: "MacBook Air M3", ModelType: "MacBookAir15,1", Board: "Mac-"},
				// iMac 系列
				{Model: "iMac 24 M1", ModelType: "iMac21,1", Board: "Mac-"},
				{Model: "iMac 24 M3", ModelType: "iMac24,1", Board: "Mac-"},
				// Mac mini 系列
				{Model: "Mac mini M1", ModelType: "Macmini9,1", Board: "Mac-"},
				{Model: "Mac mini M2", ModelType: "Macmini14,1", Board: "Mac-"},
			},
			"Microsoft": {
				// Surface Pro 系列（印度市场）
				{Model: "Surface Pro 8", ModelType: "SurfacePro8", Board: "Surface"},
				{Model: "Surface Pro 9", ModelType: "SurfacePro9", Board: "Surface"},
				{Model: "Surface Pro 10", ModelType: "SurfacePro10", Board: "Surface"},
				// Surface Laptop 系列
				{Model: "Surface Laptop 4", ModelType: "SurfaceLaptop4", Board: "Surface"},
				{Model: "Surface Laptop 5", ModelType: "SurfaceLaptop5", Board: "Surface"},
				{Model: "Surface Laptop Studio", ModelType: "SurfaceLaptopStudio", Board: "Surface"},
				{Model: "Surface Laptop Studio 2", ModelType: "SurfaceLaptopStudio2", Board: "Surface"},
				// Surface Book 系列
				{Model: "Surface Book 3", ModelType: "SurfaceBook3", Board: "Surface"},
				// Surface Go 系列
				{Model: "Surface Go 3", ModelType: "SurfaceGo3", Board: "Surface"},
				{Model: "Surface Go 4", ModelType: "SurfaceGo4", Board: "Surface"},
			},
			"Dell": {
				// XPS 系列（印度市场主流）
				{Model: "XPS 13 9310", ModelType: "XPS139310", Board: "Dell"},
				{Model: "XPS 13 9320", ModelType: "XPS139320", Board: "Dell"},
				{Model: "XPS 13 9330", ModelType: "XPS139330", Board: "Dell"},
				{Model: "XPS 13 Plus", ModelType: "XPS13Plus", Board: "Dell"},
				{Model: "XPS 15 9510", ModelType: "XPS159510", Board: "Dell"},
				{Model: "XPS 15 9520", ModelType: "XPS159520", Board: "Dell"},
				{Model: "XPS 15 9530", ModelType: "XPS159530", Board: "Dell"},
				{Model: "XPS 17 9710", ModelType: "XPS179710", Board: "Dell"},
				{Model: "XPS 17 9720", ModelType: "XPS179720", Board: "Dell"},
				// Inspiron 系列
				{Model: "Inspiron 14 3000", ModelType: "Inspiron143000", Board: "Dell"},
				{Model: "Inspiron 14 5000", ModelType: "Inspiron145000", Board: "Dell"},
				{Model: "Inspiron 15 3000", ModelType: "Inspiron153000", Board: "Dell"},
				{Model: "Inspiron 15 5000", ModelType: "Inspiron155000", Board: "Dell"},
				{Model: "Inspiron 16 Plus", ModelType: "Inspiron16Plus", Board: "Dell"},
				// Latitude 系列
				{Model: "Latitude 5430", ModelType: "Latitude5430", Board: "Dell"},
				{Model: "Latitude 5530", ModelType: "Latitude5530", Board: "Dell"},
				{Model: "Latitude 7430", ModelType: "Latitude7430", Board: "Dell"},
				{Model: "Latitude 7530", ModelType: "Latitude7530", Board: "Dell"},
				// Vostro 系列
				{Model: "Vostro 14 3000", ModelType: "Vostro143000", Board: "Dell"},
				{Model: "Vostro 15 3000", ModelType: "Vostro153000", Board: "Dell"},
				{Model: "Vostro 14 5000", ModelType: "Vostro145000", Board: "Dell"},
			},
			"HP": {
				// Spectre 系列（印度市场）
				{Model: "Spectre x360 13", ModelType: "Spectre13", Board: "HP"},
				{Model: "Spectre x360 14", ModelType: "Spectre14", Board: "HP"},
				{Model: "Spectre x360 15", ModelType: "Spectre15", Board: "HP"},
				{Model: "Spectre x360 16", ModelType: "Spectre16", Board: "HP"},
				// Envy 系列
				{Model: "Envy x360 13", ModelType: "Envy13", Board: "HP"},
				{Model: "Envy x360 15", ModelType: "Envy15", Board: "HP"},
				{Model: "Envy 13", ModelType: "Envy13", Board: "HP"},
				{Model: "Envy 15", ModelType: "Envy15", Board: "HP"},
				// Pavilion 系列
				{Model: "Pavilion 14", ModelType: "Pavilion14", Board: "HP"},
				{Model: "Pavilion 15", ModelType: "Pavilion15", Board: "HP"},
				{Model: "Pavilion Plus 14", ModelType: "PavilionPlus14", Board: "HP"},
				{Model: "Pavilion Gaming 15", ModelType: "PavilionGaming15", Board: "HP"},
				// EliteBook 系列
				{Model: "EliteBook 840 G8", ModelType: "EliteBook840G8", Board: "HP"},
				{Model: "EliteBook 840 G9", ModelType: "EliteBook840G9", Board: "HP"},
				{Model: "EliteBook 850 G8", ModelType: "EliteBook850G8", Board: "HP"},
				{Model: "EliteBook 850 G9", ModelType: "EliteBook850G9", Board: "HP"},
			},
			"Lenovo": {
				// ThinkPad 系列（印度市场主流）
				{Model: "ThinkPad X1 Carbon Gen 9", ModelType: "ThinkPadX1CarbonG9", Board: "Lenovo"},
				{Model: "ThinkPad X1 Carbon Gen 10", ModelType: "ThinkPadX1CarbonG10", Board: "Lenovo"},
				{Model: "ThinkPad X1 Carbon Gen 11", ModelType: "ThinkPadX1CarbonG11", Board: "Lenovo"},
				{Model: "ThinkPad T14 Gen 2", ModelType: "ThinkPadT14G2", Board: "Lenovo"},
				{Model: "ThinkPad T14 Gen 3", ModelType: "ThinkPadT14G3", Board: "Lenovo"},
				{Model: "ThinkPad T14 Gen 4", ModelType: "ThinkPadT14G4", Board: "Lenovo"},
				{Model: "ThinkPad T16 Gen 1", ModelType: "ThinkPadT16G1", Board: "Lenovo"},
				{Model: "ThinkPad T16 Gen 2", ModelType: "ThinkPadT16G2", Board: "Lenovo"},
				{Model: "ThinkPad E14 Gen 2", ModelType: "ThinkPadE14G2", Board: "Lenovo"},
				{Model: "ThinkPad E14 Gen 3", ModelType: "ThinkPadE14G3", Board: "Lenovo"},
				{Model: "ThinkPad E14 Gen 4", ModelType: "ThinkPadE14G4", Board: "Lenovo"},
				{Model: "ThinkPad P14s Gen 2", ModelType: "ThinkPadP14sG2", Board: "Lenovo"},
				{Model: "ThinkPad P14s Gen 3", ModelType: "ThinkPadP14sG3", Board: "Lenovo"},
				{Model: "ThinkPad P16s Gen 1", ModelType: "ThinkPadP16sG1", Board: "Lenovo"},
				{Model: "ThinkPad X1 Yoga Gen 7", ModelType: "ThinkPadX1YogaG7", Board: "Lenovo"},
				{Model: "ThinkPad X1 Yoga Gen 8", ModelType: "ThinkPadX1YogaG8", Board: "Lenovo"},
				// IdeaPad 系列
				{Model: "IdeaPad 3", ModelType: "IdeaPad3", Board: "Lenovo"},
				{Model: "IdeaPad 5", ModelType: "IdeaPad5", Board: "Lenovo"},
				{Model: "IdeaPad Flex 5", ModelType: "IdeaPadFlex5", Board: "Lenovo"},
				{Model: "IdeaPad Slim 3", ModelType: "IdeaPadSlim3", Board: "Lenovo"},
				{Model: "IdeaPad Slim 5", ModelType: "IdeaPadSlim5", Board: "Lenovo"},
				{Model: "IdeaPad Gaming 3", ModelType: "IdeaPadGaming3", Board: "Lenovo"},
				// Yoga 系列
				{Model: "Yoga 7i", ModelType: "Yoga7i", Board: "Lenovo"},
				{Model: "Yoga 9i", ModelType: "Yoga9i", Board: "Lenovo"},
				{Model: "Yoga Slim 7", ModelType: "YogaSlim7", Board: "Lenovo"},
				{Model: "Yoga Slim 7 Pro", ModelType: "YogaSlim7Pro", Board: "Lenovo"},
				// Legion 系列
				{Model: "Legion 5", ModelType: "Legion5", Board: "Lenovo"},
				{Model: "Legion 5 Pro", ModelType: "Legion5Pro", Board: "Lenovo"},
				{Model: "Legion 7", ModelType: "Legion7", Board: "Lenovo"},
			},
			"Acer": {
				// Aspire 系列（印度市场主流）
				{Model: "Aspire 3", ModelType: "Aspire3", Board: "Acer"},
				{Model: "Aspire 5", ModelType: "Aspire5", Board: "Acer"},
				{Model: "Aspire 7", ModelType: "Aspire7", Board: "Acer"},
				// Swift 系列
				{Model: "Swift 3", ModelType: "Swift3", Board: "Acer"},
				{Model: "Swift 5", ModelType: "Swift5", Board: "Acer"},
				{Model: "Swift X", ModelType: "SwiftX", Board: "Acer"},
				// Predator 系列
				{Model: "Predator Helios 300", ModelType: "PredatorHelios300", Board: "Acer"},
				{Model: "Predator Triton 300", ModelType: "PredatorTriton300", Board: "Acer"},
				// Nitro 系列
				{Model: "Nitro 5", ModelType: "Nitro5", Board: "Acer"},
				{Model: "Nitro 7", ModelType: "Nitro7", Board: "Acer"},
			},
			"ASUS": {
				// ZenBook 系列（印度市场）
				{Model: "ZenBook 13", ModelType: "ZenBook13", Board: "ASUS"},
				{Model: "ZenBook 14", ModelType: "ZenBook14", Board: "ASUS"},
				{Model: "ZenBook 15", ModelType: "ZenBook15", Board: "ASUS"},
				// VivoBook 系列
				{Model: "VivoBook 14", ModelType: "VivoBook14", Board: "ASUS"},
				{Model: "VivoBook 15", ModelType: "VivoBook15", Board: "ASUS"},
				{Model: "VivoBook S14", ModelType: "VivoBookS14", Board: "ASUS"},
				{Model: "VivoBook S15", ModelType: "VivoBookS15", Board: "ASUS"},
				// ROG 系列
				{Model: "ROG Zephyrus G14", ModelType: "ROGZephyrusG14", Board: "ASUS"},
				{Model: "ROG Zephyrus G15", ModelType: "ROGZephyrusG15", Board: "ASUS"},
				{Model: "ROG Strix G15", ModelType: "ROGStrixG15", Board: "ASUS"},
				{Model: "ROG Strix G16", ModelType: "ROGStrixG16", Board: "ASUS"},
				// TUF Gaming 系列
				{Model: "TUF Gaming F15", ModelType: "TUFF15", Board: "ASUS"},
				{Model: "TUF Gaming A15", ModelType: "TUFA15", Board: "ASUS"},
			},
		},
	}

	_ = fingerprint.RegisterRegion(config)
}
