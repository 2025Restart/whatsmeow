// Copyright (c) 2024
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sqlstore

// Region 表示设备指纹生成的地区
type Region int

const (
	// Region_None 表示未配置地区，不启用指纹功能
	Region_None Region = iota
	// Region_IN 印度
	Region_IN
	// Region_BR 巴西
	Region_BR
	// 未来可扩展：Region_MX, Region_US 等
)

// String 返回地区的 ISO 3166-1 alpha-2 代码
func (r Region) String() string {
	switch r {
	case Region_IN:
		return "IN"
	case Region_BR:
		return "BR"
	default:
		return ""
	}
}

// IsValid 检查地区是否有效（非 None）
func (r Region) IsValid() bool {
	return r != Region_None
}
