// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fingerprint

import (
	"fmt"
	"sync"
)

var (
	registry      = make(map[string]*RegionConfig)
	registryLock  sync.RWMutex
	defaultRegion *RegionConfig
)

// RegisterRegion 注册地区配置
func RegisterRegion(config *RegionConfig) error {
	if config == nil {
		return fmt.Errorf("region config cannot be nil")
	}
	if config.Code == "" {
		return fmt.Errorf("region code cannot be empty")
	}

	registryLock.Lock()
	defer registryLock.Unlock()

	// 验证配置
	if err := validateRegionConfig(config); err != nil {
		return fmt.Errorf("invalid region config for %s: %w", config.Code, err)
	}

	registry[config.Code] = config
	return nil
}

// GetRegionConfig 获取地区配置
func GetRegionConfig(regionCode string) *RegionConfig {
	registryLock.RLock()
	defer registryLock.RUnlock()

	if config, ok := registry[regionCode]; ok {
		return config
	}

	// 返回默认配置（向后兼容）
	return defaultRegion
}

// ListRegions 列出所有已注册的地区
func ListRegions() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()

	regions := make([]string, 0, len(registry))
	for code := range registry {
		regions = append(regions, code)
	}
	return regions
}

// SetDefaultRegion 设置默认地区配置
func SetDefaultRegion(config *RegionConfig) error {
	if err := validateRegionConfig(config); err != nil {
		return fmt.Errorf("invalid default region config: %w", err)
	}
	defaultRegion = config
	return nil
}

// validateRegionConfig 验证地区配置
func validateRegionConfig(config *RegionConfig) error {
	// 验证语言权重总和
	var langWeightSum float64
	for _, lang := range config.Languages {
		if lang.Weight < 0 || lang.Weight > 1 {
			return fmt.Errorf("language weight must be between 0 and 1")
		}
		langWeightSum += lang.Weight
	}
	if len(config.Languages) > 0 && langWeightSum > 1.01 { // 允许小的浮点误差
		return fmt.Errorf("language weights sum should be <= 1.0, got %.2f", langWeightSum)
	}

	// 验证平台分布权重总和
	var platformWeightSum float64
	for _, dist := range config.DeviceDistribution {
		if dist.Weight < 0 || dist.Weight > 1 {
			return fmt.Errorf("platform weight must be between 0 and 1")
		}
		platformWeightSum += dist.Weight
	}
	if len(config.DeviceDistribution) > 0 && platformWeightSum > 1.01 {
		return fmt.Errorf("platform weights sum should be <= 1.0, got %.2f", platformWeightSum)
	}

	// 验证移动网络权重总和
	var networkWeightSum float64
	for _, net := range config.MobileNetworks {
		if net.Weight < 0 || net.Weight > 1 {
			return fmt.Errorf("network weight must be between 0 and 1")
		}
		networkWeightSum += net.Weight
	}
	if len(config.MobileNetworks) > 0 && networkWeightSum > 1.01 {
		return fmt.Errorf("network weights sum should be <= 1.0, got %.2f", networkWeightSum)
	}

	return nil
}
