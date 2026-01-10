// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package store

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/libsignal/ecc"

	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/types"
)

// 设备指纹相关常量（统一管理）
const (
	// DefaultBrowserName 默认浏览器名称
	DefaultBrowserName = "Google Chrome"

	// DefaultOSName 默认操作系统名称（用于显示）
	DefaultOSName = "Windows"

	// DefaultDevicePropsOs 默认 DeviceProps.Os 值
	// 注意：当 PlatformType 是浏览器类型时，WhatsApp 会自动添加浏览器名称前缀
	// 因此 Os 字段应该只包含操作系统名称，避免重复显示
	// 格式：操作系统名称
	// 示例：Windows, Mac OS, Linux
	DefaultDevicePropsOs = DefaultOSName
)

// WAVersionContainer is a container for a WhatsApp web version number.
type WAVersionContainer [3]uint32

// ParseVersion parses a version string (three dot-separated numbers) into a WAVersionContainer.
func ParseVersion(version string) (parsed WAVersionContainer, err error) {
	var part1, part2, part3 int
	if parts := strings.Split(version, "."); len(parts) != 3 {
		err = fmt.Errorf("'%s' doesn't contain three dot-separated parts", version)
	} else if part1, err = strconv.Atoi(parts[0]); err != nil {
		err = fmt.Errorf("first part of '%s' is not a number: %w", version, err)
	} else if part2, err = strconv.Atoi(parts[1]); err != nil {
		err = fmt.Errorf("second part of '%s' is not a number: %w", version, err)
	} else if part3, err = strconv.Atoi(parts[2]); err != nil {
		err = fmt.Errorf("third part of '%s' is not a number: %w", version, err)
	} else {
		parsed = WAVersionContainer{uint32(part1), uint32(part2), uint32(part3)}
	}
	return
}

func (vc WAVersionContainer) LessThan(other WAVersionContainer) bool {
	return vc[0] < other[0] ||
		(vc[0] == other[0] && vc[1] < other[1]) ||
		(vc[0] == other[0] && vc[1] == other[1] && vc[2] < other[2])
}

// IsZero returns true if the version is zero.
func (vc WAVersionContainer) IsZero() bool {
	return vc == [3]uint32{0, 0, 0}
}

// String returns the version number as a dot-separated string.
func (vc WAVersionContainer) String() string {
	parts := make([]string, len(vc))
	for i, part := range vc {
		parts[i] = strconv.Itoa(int(part))
	}
	return strings.Join(parts, ".")
}

// Hash returns the md5 hash of the String representation of this version.
func (vc WAVersionContainer) Hash() [16]byte {
	return md5.Sum([]byte(vc.String()))
}

func (vc WAVersionContainer) ProtoAppVersion() *waWa6.ClientPayload_UserAgent_AppVersion {
	return &waWa6.ClientPayload_UserAgent_AppVersion{
		Primary:   &vc[0],
		Secondary: &vc[1],
		Tertiary:  &vc[2],
	}
}

// waVersion is the WhatsApp web client version
var waVersion = WAVersionContainer{2, 3000, 1031788782}

// waVersionHash is the md5 hash of a dot-separated waVersion
var waVersionHash [16]byte

func init() {
	waVersionHash = waVersion.Hash()
}

// GetWAVersion gets the current WhatsApp web client version.
func GetWAVersion() WAVersionContainer {
	return waVersion
}

// SetWAVersion sets the current WhatsApp web client version.
//
// In general, you should keep the library up-to-date instead of using this,
// as there may be code changes that are necessary too (like protobuf schema changes).
func SetWAVersion(version WAVersionContainer) {
	if version.IsZero() {
		return
	}
	waVersion = version
	waVersionHash = version.Hash()
	// 同步更新模板中的 AppVersion，确保后续 Clone 得到的是最新版本
	if BaseClientPayload.UserAgent != nil {
		BaseClientPayload.UserAgent.AppVersion = waVersion.ProtoAppVersion()
	}
}

var BaseClientPayload = &waWa6.ClientPayload{
	UserAgent: &waWa6.ClientPayload_UserAgent{
		Platform:       waWa6.ClientPayload_UserAgent_WEB.Enum(),
		ReleaseChannel: waWa6.ClientPayload_UserAgent_RELEASE.Enum(),
		AppVersion:     waVersion.ProtoAppVersion(),
		Mcc:            proto.String("000"),
		Mnc:            proto.String("000"),
		OsVersion:      proto.String("0.1"),
		Manufacturer:   proto.String("Microsoft"),
		Device:         proto.String("Desktop"),
		OsBuildNumber:  proto.String("0.1"),

		LocaleLanguageIso6391:       proto.String("en"),
		LocaleCountryIso31661Alpha2: proto.String("US"),
	},
	WebInfo: &waWa6.ClientPayload_WebInfo{
		WebSubPlatform: waWa6.ClientPayload_WebInfo_WEB_BROWSER.Enum(),
	},
	ConnectType:   waWa6.ClientPayload_WIFI_UNKNOWN.Enum(),
	ConnectReason: waWa6.ClientPayload_USER_ACTIVATED.Enum(),
}

var DeviceProps = &waCompanionReg.DeviceProps{
	Os: proto.String(DefaultDevicePropsOs), // 使用统一管理的默认值
	Version: &waCompanionReg.DeviceProps_AppVersion{
		Primary:   proto.Uint32(0),
		Secondary: proto.Uint32(1),
		Tertiary:  proto.Uint32(0),
	},
	HistorySyncConfig: &waCompanionReg.DeviceProps_HistorySyncConfig{
		StorageQuotaMb:                           proto.Uint32(10240),
		InlineInitialPayloadInE2EeMsg:            proto.Bool(true),
		RecentSyncDaysLimit:                      nil,
		SupportCallLogHistory:                    proto.Bool(false),
		SupportBotUserAgentChatHistory:           proto.Bool(true),
		SupportCagReactionsAndPolls:              proto.Bool(true),
		SupportBizHostedMsg:                      proto.Bool(true),
		SupportRecentSyncChunkMessageCountTuning: proto.Bool(true),
		SupportHostedGroupMsg:                    proto.Bool(true),
		SupportFbidBotChatHistory:                proto.Bool(true),
		SupportAddOnHistorySyncMigration:         nil,
		SupportMessageAssociation:                proto.Bool(true),
		SupportGroupHistory:                      proto.Bool(false),
		OnDemandReady:                            nil,
		SupportGuestChat:                         nil,
		CompleteOnDemandReady:                    nil,
		ThumbnailSyncDaysLimit:                   nil,
	},
	PlatformType:    waCompanionReg.DeviceProps_CHROME.Enum(), // 默认 Chrome 以显示浏览器图标
	RequireFullSync: proto.Bool(false),
}

func SetOSInfo(name string, version [3]uint32) {
	DeviceProps.Os = &name
	DeviceProps.Version.Primary = &version[0]
	DeviceProps.Version.Secondary = &version[1]
	DeviceProps.Version.Tertiary = &version[2]
	BaseClientPayload.UserAgent.OsVersion = proto.String(fmt.Sprintf("%d.%d.%d", version[0], version[1], version[2]))
	BaseClientPayload.UserAgent.OsBuildNumber = BaseClientPayload.UserAgent.OsVersion
}

func (device *Device) getHistorySyncProps() *waCompanionReg.DeviceProps {
	props := proto.Clone(DeviceProps).(*waCompanionReg.DeviceProps)

	// 如果 DeviceProps.Version 还是默认的 0.1.0，同步为当前的 waVersion
	// 这解决了用户提到的“底层有 GetLatestVersion 能同步 version”但 Props 未同步的问题
	if props.Version != nil && props.Version.GetPrimary() == 0 && props.Version.GetSecondary() == 1 {
		props.Version = &waCompanionReg.DeviceProps_AppVersion{
			Primary:   proto.Uint32(waVersion[0]),
			Secondary: proto.Uint32(waVersion[1]),
			Tertiary:  proto.Uint32(waVersion[2]),
		}
	}

	// L4: 客户端属性随机化（基于设备 ID 的幂等随机）
	var seed int64
	if device.ID != nil {
		seed = int64(device.ID.UserInt())
	} else {
		seed = int64(device.RegistrationID)
	}
	r := rand.New(rand.NewSource(seed))

	// 确保 HistorySyncConfig 存在以避免 nil panic
	if props.HistorySyncConfig == nil {
		props.HistorySyncConfig = &waCompanionReg.DeviceProps_HistorySyncConfig{}
	}

	// 随机微调存储配额 (10GB - 20GB)
	quota := 10240 + uint32(r.Intn(10240))
	props.HistorySyncConfig.StorageQuotaMb = proto.Uint32(quota)

	// 随机决定是否支持某些次要特性
	props.HistorySyncConfig.SupportCallLogHistory = proto.Bool(r.Intn(2) == 0)

	return props
}

func (device *Device) getRegistrationPayload() *waWa6.ClientPayload {
	payload := proto.Clone(BaseClientPayload).(*waWa6.ClientPayload)
	regID := make([]byte, 4)
	binary.BigEndian.PutUint32(regID, device.RegistrationID)
	preKeyID := make([]byte, 4)
	binary.BigEndian.PutUint32(preKeyID, device.SignedPreKey.KeyID)
	// 确保 DeviceProps.Os 不是非官方标识（双重保障）
	if DeviceProps.Os != nil && *DeviceProps.Os == "whatsmeow" {
		// 使用统一管理的默认值
		// 注意：如果启用了指纹功能，这个值会被 fingerprint.ApplyFingerprint 覆盖
		DeviceProps.Os = proto.String(DefaultDevicePropsOs)
	}
	deviceProps, _ := proto.Marshal(device.getHistorySyncProps())
	payload.DevicePairingData = &waWa6.ClientPayload_DevicePairingRegistrationData{
		ERegid:      regID,
		EKeytype:    []byte{ecc.DjbType},
		EIdent:      device.IdentityKey.Pub[:],
		ESkeyID:     preKeyID[1:],
		ESkeyVal:    device.SignedPreKey.Pub[:],
		ESkeySig:    device.SignedPreKey.Signature[:],
		BuildHash:   waVersionHash[:],
		DeviceProps: deviceProps,
	}
	payload.Passive = proto.Bool(false)
	payload.Pull = proto.Bool(false)
	return sanitizeClientPayload(payload)
}

func (device *Device) getLoginPayload() *waWa6.ClientPayload {
	payload := proto.Clone(BaseClientPayload).(*waWa6.ClientPayload)
	payload.Username = proto.Uint64(device.ID.UserInt())
	payload.Device = proto.Uint32(uint32(device.ID.Device))
	payload.Passive = proto.Bool(true)
	payload.Pull = proto.Bool(true)
	payload.LidDbMigrated = proto.Bool(true)
	return sanitizeClientPayload(payload)
}

// sanitizeClientPayload 最后的特征清理（防火墙）
func sanitizeClientPayload(payload *waWa6.ClientPayload) *waWa6.ClientPayload {
	if payload == nil {
		return nil
	}
	// L5: 彻底禁止 whatsmeow 特征泄露
	if payload.UserAgent != nil {
		if payload.UserAgent.Manufacturer != nil && *payload.UserAgent.Manufacturer == "whatsmeow" {
			payload.UserAgent.Manufacturer = proto.String("Microsoft")
		}
		if payload.UserAgent.Device != nil && *payload.UserAgent.Device == "whatsmeow" {
			payload.UserAgent.Device = proto.String("Desktop")
		}
	}
	return payload
}

func (device *Device) GetClientPayload() *waWa6.ClientPayload {
	if device.ID != nil {
		if *device.ID == types.EmptyJID {
			panic(fmt.Errorf("GetClientPayload called with empty JID"))
		}
		return device.getLoginPayload()
	} else {
		return device.getRegistrationPayload()
	}
}
