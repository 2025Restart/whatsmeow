// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package whatsmeow

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/libsignal/ecc"
	"google.golang.org/protobuf/proto"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/util/keys"
)

var (
	AdvAccountSignaturePrefix = []byte{6, 0}
	AdvDeviceSignaturePrefix  = []byte{6, 1}

	AdvHostedAccountSignaturePrefix = []byte{6, 5}
	AdvHostedDeviceSignaturePrefix  = []byte{6, 6}
)

func (cli *Client) handleIQ(ctx context.Context, node *waBinary.Node) {
	children := node.GetChildren()
	if len(children) != 1 || node.Attrs["from"] != types.ServerJID {
		return
	}
	switch children[0].Tag {
	case "pair-device":
		cli.handlePairDevice(ctx, node)
	case "pair-success":
		cli.handlePairSuccess(ctx, node)
	}
}

func (cli *Client) handlePairDevice(ctx context.Context, node *waBinary.Node) {
	pairDevice := node.GetChildByTag("pair-device")
	err := cli.sendNode(ctx, waBinary.Node{
		Tag: "iq",
		Attrs: waBinary.Attrs{
			"to":   node.Attrs["from"],
			"id":   node.Attrs["id"],
			"type": "result",
		},
	})
	if err != nil {
		cli.Log.Warnf("Failed to send acknowledgement for pair-device request: %v", err)
	}

	evt := &events.QR{Codes: make([]string, 0, len(pairDevice.GetChildren()))}
	for i, child := range pairDevice.GetChildren() {
		if child.Tag != "ref" {
			cli.Log.Warnf("pair-device node contains unexpected child tag %s at index %d", child.Tag, i)
			continue
		}
		content, ok := child.Content.([]byte)
		if !ok {
			cli.Log.Warnf("pair-device node contains unexpected child content type %T at index %d", child, i)
			continue
		}
		evt.Codes = append(evt.Codes, cli.makeQRData(string(content)))
	}

	cli.dispatchEvent(evt)
}

func (cli *Client) makeQRData(ref string) string {
	noise := base64.StdEncoding.EncodeToString(cli.Store.NoiseKey.Pub[:])
	identity := base64.StdEncoding.EncodeToString(cli.Store.IdentityKey.Pub[:])
	adv := base64.StdEncoding.EncodeToString(cli.Store.AdvSecretKey)
	return strings.Join([]string{ref, noise, identity, adv}, ",")
}

func (cli *Client) handlePairSuccess(ctx context.Context, node *waBinary.Node) {
	id := node.Attrs["id"].(string)
	pairSuccess := node.GetChildByTag("pair-success")

	deviceIdentityBytes, _ := pairSuccess.GetChildByTag("device-identity").Content.([]byte)
	businessName, _ := pairSuccess.GetChildByTag("biz").Attrs["name"].(string)
	jid, _ := pairSuccess.GetChildByTag("device").Attrs["jid"].(types.JID)
	lid, _ := pairSuccess.GetChildByTag("device").Attrs["lid"].(types.JID)
	platform, _ := pairSuccess.GetChildByTag("platform").Attrs["name"].(string)

	cli.Log.Debugf("[PairSuccess] Server provided JID: %s, LID: %s (empty=%v)", jid, lid, lid.IsEmpty())
	if lid.IsEmpty() {
		cli.Log.Warnf("[PairSuccess] No LID provided in pair-success node for JID: %s", jid)
	}

	go func() {
		err := cli.handlePair(ctx, deviceIdentityBytes, id, businessName, platform, jid, lid)
		if err != nil {
			cli.Log.Errorf("Failed to pair device: %v", err)
			cli.Disconnect()
			cli.dispatchEvent(&events.PairError{ID: jid, LID: lid, BusinessName: businessName, Platform: platform, Error: err})
		} else {
			cli.Log.Infof("Successfully paired %s", cli.Store.ID)
			go cli.sendUnifiedSession(ctx)
			cli.dispatchEvent(&events.PairSuccess{ID: jid, LID: lid, BusinessName: businessName, Platform: platform})
		}
	}()
}

func (cli *Client) handlePair(ctx context.Context, deviceIdentityBytes []byte, reqID, businessName, platform string, jid, lid types.JID) error {
	var deviceIdentityContainer waAdv.ADVSignedDeviceIdentityHMAC
	err := proto.Unmarshal(deviceIdentityBytes, &deviceIdentityContainer)
	if err != nil {
		cli.sendPairError(ctx, reqID, 500, "internal-error")
		return &PairProtoError{"failed to parse device identity container in pair success message", err}
	}

	h := hmac.New(sha256.New, cli.Store.AdvSecretKey)
	if deviceIdentityContainer.GetAccountType() == waAdv.ADVEncryptionType_HOSTED {
		h.Write(AdvHostedAccountSignaturePrefix)
		//cli.Store.IsHosted = true
	}
	h.Write(deviceIdentityContainer.Details)

	if !bytes.Equal(h.Sum(nil), deviceIdentityContainer.HMAC) {
		cli.Log.Warnf("Invalid HMAC from pair success message")
		cli.sendPairError(ctx, reqID, 401, "hmac-mismatch")
		return ErrPairInvalidDeviceIdentityHMAC
	}

	var deviceIdentity waAdv.ADVSignedDeviceIdentity
	err = proto.Unmarshal(deviceIdentityContainer.Details, &deviceIdentity)
	if err != nil {
		cli.sendPairError(ctx, reqID, 500, "internal-error")
		return &PairProtoError{"failed to parse signed device identity in pair success message", err}
	}

	var deviceIdentityDetails waAdv.ADVDeviceIdentity
	err = proto.Unmarshal(deviceIdentity.Details, &deviceIdentityDetails)
	if err != nil {
		cli.sendPairError(ctx, reqID, 500, "internal-error")
		return &PairProtoError{"failed to parse device identity details in pair success message", err}
	}

	if !verifyAccountSignature(&deviceIdentity, cli.Store.IdentityKey, deviceIdentityDetails.GetDeviceType() == waAdv.ADVEncryptionType_HOSTED) {
		cli.sendPairError(ctx, reqID, 401, "signature-mismatch")
		return ErrPairInvalidDeviceSignature
	}

	deviceIdentity.DeviceSignature = generateDeviceSignature(&deviceIdentity, cli.Store.IdentityKey)[:]

	if cli.PrePairCallback != nil && !cli.PrePairCallback(jid, platform, businessName) {
		cli.sendPairError(ctx, reqID, 500, "internal-error")
		return ErrPairRejectedLocally
	}

	cli.Store.Account = proto.Clone(&deviceIdentity).(*waAdv.ADVSignedDeviceIdentity)

	mainDeviceLID := lid
	mainDeviceLID.Device = 0
	mainDeviceIdentity := *(*[32]byte)(deviceIdentity.AccountSignatureKey)
	deviceIdentity.AccountSignatureKey = nil

	selfSignedDeviceIdentity, err := proto.Marshal(&deviceIdentity)
	if err != nil {
		cli.sendPairError(ctx, reqID, 500, "internal-error")
		return &PairProtoError{"failed to marshal self-signed device identity", err}
	}

	cli.Store.ID = &jid
	cli.Store.LID = lid
	cli.Store.BusinessName = businessName
	cli.Store.Platform = platform
	err = cli.Store.Save(ctx)
	if err != nil {
		cli.sendPairError(ctx, reqID, 500, "internal-error")
		return &PairDatabaseError{"failed to save device store", err}
	}
	if !lid.IsEmpty() {
		cli.Log.Infof("[PairSuccess] Saved LID from server: %s (JID: %s)", lid, jid)
	} else {
		cli.Log.Warnf("[PairSuccess] Saved pair success but LID is empty (JID: %s)", jid)
	}
	cli.StoreLIDPNMapping(ctx, lid, jid)

	// 保存临时指纹到数据库（如果存在）
	// 优先从电话号码 JID 读取临时指纹，如果没有则使用内存中的临时指纹
	if container, ok := cli.Store.Container.(*sqlstore.Container); ok {
		var tempJID types.JID
		if cli.phoneLinkingCache != nil && !cli.phoneLinkingCache.jid.IsEmpty() {
			tempJID = cli.phoneLinkingCache.jid
		}

		if cli.Log != nil {
			cli.Log.Infof("[Fingerprint] Pair success: migrating temporary fingerprint to JID %s", jid.User)
		}

		// 尝试从电话号码 JID 读取临时指纹
		var pendingFP *store.DeviceFingerprint
		if !tempJID.IsEmpty() {
			var err error
			if cli.Log != nil {
				cli.Log.Infof("[Fingerprint] Pair success: attempting to load temporary fingerprint from phone JID %s", tempJID.User)
			}
			pendingFP, err = container.GetFingerprint(context.Background(), tempJID)
			if err != nil {
				cli.Log.Warnf("[Fingerprint] Failed to get temporary fingerprint from %s: %v", tempJID.User, err)
			} else if pendingFP != nil {
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] Found temporary fingerprint from phone JID %s", tempJID.User)
				}
			}
		}

		// 如果数据库中没有，尝试使用内存中的临时指纹
		if pendingFP == nil {
			cli.pendingFingerprintLock.Lock()
			if cli.pendingFingerprint != nil {
				pendingFP = cli.pendingFingerprint
				cli.pendingFingerprintLock.Unlock()
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] Using pending fingerprint from memory (not found in database)")
				}
			} else {
				cli.pendingFingerprintLock.Unlock()
				if cli.Log != nil {
					cli.Log.Warnf("[Fingerprint] No pending fingerprint found (neither in database nor memory)")
				}
			}
		}

		// 保存临时指纹到实际的 JID
		if pendingFP != nil {
			if cli.Log != nil {
				cli.Log.Infof("[Fingerprint] Migrating fingerprint to JID %s (fingerprint: %s %s, MCC: %s, MNC: %s)",
					jid.User, pendingFP.Manufacturer, pendingFP.Device, pendingFP.Mcc, pendingFP.Mnc)
			}
			// 配对成功后处理会话地理信息缓存
			// 优先使用业务层已设置的地理信息，确保配对和连接成功的指纹一致
			sessionGeo := cli.getSessionGeoCache()
			// 先复制指纹结构体，避免并发修改
			fpCopy := *pendingFP
			if sessionGeo != nil && sessionGeo.Locked {
				// 业务层已设置，确保指纹中的地理信息与业务层一致
				if fpCopy.LocaleCountry != sessionGeo.Country || fpCopy.LocaleLanguage != sessionGeo.Language {
					if cli.Log != nil {
						cli.Log.Infof("[Fingerprint] Updating fingerprint geo info to match session cache: Country=%s->%s, Language=%s->%s",
							fpCopy.LocaleCountry, sessionGeo.Country, fpCopy.LocaleLanguage, sessionGeo.Language)
					}
					fpCopy.LocaleCountry = sessionGeo.Country
					fpCopy.LocaleLanguage = sessionGeo.Language
				}
			} else if fpCopy.LocaleCountry != "" && fpCopy.LocaleLanguage != "" {
				// 业务层未设置，从指纹读取作为兜底
				timezone := getTimezoneByCountry(fpCopy.LocaleCountry)
				cli.SetUserLoginGeoInfo(fpCopy.LocaleCountry, timezone, fpCopy.LocaleLanguage)
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] Set session geo cache from fingerprint: Country=%s, Timezone=%s, Language=%s",
						fpCopy.LocaleCountry, timezone, fpCopy.LocaleLanguage)
				}
			}
			// 捕获 tempJID 到 goroutine 中，避免闭包问题
			tempJIDForDelete := tempJID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if saveErr := container.PutFingerprint(ctx, jid, &fpCopy); saveErr != nil {
				if cli.Log != nil {
					cli.Log.Warnf("[Fingerprint] Failed to save pending fingerprint after pairing for %s: %v", jid.User, saveErr)
				}
				// 保存失败时不删除临时指纹，保留供重试使用
			} else {
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] Successfully migrated temporary fingerprint to JID %s after pairing", jid.User)
				}
				// 清除内存中的临时指纹（配对成功后不再需要）
				cli.pendingFingerprintLock.Lock()
				cli.pendingFingerprint = nil
				cli.pendingFingerprintSaved.Store(false)
				cli.pendingFingerprintLock.Unlock()
				// 删除 tempJID 的临时指纹（配对成功后不再需要，避免数据库残留）
				// 注意：删除操作在保存成功后进行，确保主指纹已保存
				if !tempJIDForDelete.IsEmpty() {
					if delErr := container.DeleteFingerprint(ctx, tempJIDForDelete); delErr != nil {
						if cli.Log != nil {
							cli.Log.Warnf("[Fingerprint] Failed to delete temporary fingerprint for %s: %v (non-critical, may retry later)", tempJIDForDelete.User, delErr)
						}
						// 删除失败不影响配对流程，后续可以通过清理任务处理
					} else {
						if cli.Log != nil {
							cli.Log.Infof("[Fingerprint] Successfully deleted temporary fingerprint for %s after pairing", tempJIDForDelete.User)
						}
					}
				} else if cli.Log != nil {
					cli.Log.Debugf("[Fingerprint] Skipping temporary fingerprint deletion: tempJID is empty")
				}
			}
		}()
		}
	}
	err = cli.Store.Identities.PutIdentity(ctx, mainDeviceLID.SignalAddress().String(), mainDeviceIdentity)
	if err != nil {
		_ = cli.Store.Delete(ctx)
		cli.sendPairError(ctx, reqID, 500, "internal-error")
		return &PairDatabaseError{"failed to store main device identity", err}
	}

	// Expect a disconnect after this and don't dispatch the usual Disconnected event
	cli.expectDisconnect()

	err = cli.sendNode(ctx, waBinary.Node{
		Tag: "iq",
		Attrs: waBinary.Attrs{
			"to":   types.ServerJID,
			"type": "result",
			"id":   reqID,
		},
		Content: []waBinary.Node{{
			Tag: "pair-device-sign",
			Content: []waBinary.Node{{
				Tag: "device-identity",
				Attrs: waBinary.Attrs{
					"key-index": deviceIdentityDetails.GetKeyIndex(),
				},
				Content: selfSignedDeviceIdentity,
			}},
		}},
	})
	if err != nil {
		_ = cli.Store.Delete(ctx)
		return fmt.Errorf("failed to send pairing confirmation: %w", err)
	}
	return nil
}

func concatBytes(data ...[]byte) []byte {
	length := 0
	for _, item := range data {
		length += len(item)
	}
	output := make([]byte, length)
	ptr := 0
	for _, item := range data {
		ptr += copy(output[ptr:ptr+len(item)], item)
	}
	return output
}

func verifyAccountSignature(deviceIdentity *waAdv.ADVSignedDeviceIdentity, ikp *keys.KeyPair, isHosted bool) bool {
	if len(deviceIdentity.AccountSignatureKey) != 32 || len(deviceIdentity.AccountSignature) != 64 {
		return false
	}

	signatureKey := ecc.NewDjbECPublicKey(*(*[32]byte)(deviceIdentity.AccountSignatureKey))
	signature := *(*[64]byte)(deviceIdentity.AccountSignature)

	prefix := AdvAccountSignaturePrefix
	if isHosted {
		prefix = AdvHostedAccountSignaturePrefix
	}
	message := concatBytes(prefix, deviceIdentity.Details, ikp.Pub[:])

	return ecc.VerifySignature(signatureKey, message, signature)
}

func generateDeviceSignature(deviceIdentity *waAdv.ADVSignedDeviceIdentity, ikp *keys.KeyPair) *[64]byte {
	prefix := AdvDeviceSignaturePrefix
	message := concatBytes(prefix, deviceIdentity.Details, ikp.Pub[:], deviceIdentity.AccountSignatureKey)
	sig := ecc.CalculateSignature(ecc.NewDjbECPrivateKey(*ikp.Priv), message)
	return &sig
}

func (cli *Client) sendPairError(ctx context.Context, id string, code int, text string) {
	err := cli.sendNode(ctx, waBinary.Node{
		Tag: "iq",
		Attrs: waBinary.Attrs{
			"to":   types.ServerJID,
			"type": "error",
			"id":   id,
		},
		Content: []waBinary.Node{{
			Tag: "error",
			Attrs: waBinary.Attrs{
				"code": code,
				"text": text,
			},
		}},
	})
	if err != nil {
		cli.Log.Errorf("Failed to send pair error node: %v", err)
	}
}
