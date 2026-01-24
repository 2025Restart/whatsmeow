// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package whatsmeow

import (
	"context"
	"time"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (cli *Client) handleStreamError(ctx context.Context, node *waBinary.Node) {
	cli.isLoggedIn.Store(false)
	cli.clearResponseWaiters(node)
	code, _ := node.Attrs["code"].(string)
	conflict, _ := node.GetOptionalChildByTag("conflict")
	conflictType := conflict.AttrGetter().OptionalString("type")
	switch {
	case code == "515":
		if cli.DisableLoginAutoReconnect {
			cli.Log.Infof("Got 515 code, but login autoreconnect is disabled, not reconnecting")
			cli.dispatchEvent(&events.ManualLoginReconnect{})
			return
		}
		cli.Log.Infof("Got 515 code, reconnecting...")
		go func() {
			cli.Disconnect()
			err := cli.connect(ctx)
			if err != nil {
				cli.Log.Errorf("Failed to reconnect after 515 code: %v", err)
			}
		}()
	case code == "401" && conflictType == "device_removed":
		cli.expectDisconnect()
		cli.Log.Infof("Got device removed stream error, sending LoggedOut event and deleting session")
		go cli.dispatchEvent(&events.LoggedOut{OnConnect: false, Reason: events.ConnectFailureLoggedOut})
		err := cli.Store.Delete(ctx)
		if err != nil {
			cli.Log.Warnf("Failed to delete store after device_removed error: %v", err)
		}
	case conflictType == "replaced":
		cli.expectDisconnect()
		cli.Log.Infof("Got replaced stream error, sending StreamReplaced event")
		go cli.dispatchEvent(&events.StreamReplaced{})
	case code == "503":
		// This seems to happen when the server wants to restart or something.
		// The disconnection will be emitted as an events.Disconnected and then the auto-reconnect will do its thing.
		cli.Log.Warnf("Got 503 stream error, assuming automatic reconnect will handle it")
	case cli.RefreshCAT != nil && (code == events.ConnectFailureCATInvalid.NumberString() || code == events.ConnectFailureCATExpired.NumberString()):
		cli.Log.Infof("Got %s stream error, refreshing CAT before reconnecting...", code)
		cli.socketLock.RLock()
		defer cli.socketLock.RUnlock()
		err := cli.RefreshCAT(ctx)
		if err != nil {
			cli.Log.Errorf("Failed to refresh CAT: %v", err)
			cli.expectDisconnect()
			go cli.dispatchEvent(&events.CATRefreshError{Error: err})
		}
	default:
		cli.Log.Errorf("Unknown stream error: %s", node.XMLString())
		go cli.dispatchEvent(&events.StreamError{Code: code, Raw: node})
	}
}

func (cli *Client) handleIB(ctx context.Context, node *waBinary.Node) {
	children := node.GetChildren()
	for _, child := range children {
		ag := child.AttrGetter()
		switch child.Tag {
		case "downgrade_webclient":
			go cli.dispatchEvent(&events.QRScannedWithoutMultidevice{})
		case "offline_preview":
			cli.dispatchEvent(&events.OfflineSyncPreview{
				Total:          ag.Int("count"),
				AppDataChanges: ag.Int("appdata"),
				Messages:       ag.Int("message"),
				Notifications:  ag.Int("notification"),
				Receipts:       ag.Int("receipt"),
			})
		case "offline":
			cli.dispatchEvent(&events.OfflineSyncCompleted{
				Count: ag.Int("count"),
			})
		case "dirty":
			//ts := ag.UnixTime("timestamp")
			//typ := ag.String("type") // account_sync
			//go func() {
			//	err := cli.MarkNotDirty(ctx, typ, ts)
			//	zerolog.Ctx(ctx).Debug().Err(err).Msg("Marked dirty item as clean")
			//}()
		}
	}
}

func (cli *Client) handleConnectFailure(ctx context.Context, node *waBinary.Node) {
	ag := node.AttrGetter()
	reason := events.ConnectFailureReason(ag.Int("reason"))
	message := ag.OptionalString("message")
	location := ag.OptionalString("location")
	willAutoReconnect := true
	switch {
	default:
		// By default, expect a disconnect (i.e. prevent auto-reconnect)
		cli.expectDisconnect()
		willAutoReconnect = false
	case reason == events.ConnectFailureServiceUnavailable || reason == events.ConnectFailureInternalServerError:
		// Auto-reconnect for 503s
	case reason == events.ConnectFailureCATInvalid || reason == events.ConnectFailureCATExpired:
		// Auto-reconnect when rotating CAT, lock socket to ensure refresh goes through before reconnect
		cli.socketLock.RLock()
		defer cli.socketLock.RUnlock()
	}
	if reason == 403 {
		cli.Log.Debugf(
			"Message for 403 connect failure: %s / %s (location: %s)",
			ag.OptionalString("logout_message_header"),
			ag.OptionalString("logout_message_subtext"),
			location,
		)
	}
	if reason.IsLoggedOut() {
		cli.Log.Infof("Got %s connect failure (location: %s), sending LoggedOut event and deleting session", reason, location)
		go cli.dispatchEvent(&events.LoggedOut{OnConnect: true, Reason: reason})
		err := cli.Store.Delete(ctx)
		if err != nil {
			cli.Log.Warnf("Failed to delete store after %d failure: %v", int(reason), err)
		}
	} else if reason == events.ConnectFailureTempBanned {
		cli.Log.Warnf("Temporary ban connect failure (location: %s): %s", location, node.XMLString())
		go cli.dispatchEvent(&events.TemporaryBan{
			Code:   events.TempBanReason(ag.Int("code")),
			Expire: time.Duration(ag.Int("expire")) * time.Second,
		})
	} else if reason == events.ConnectFailureClientOutdated {
		cli.Log.Errorf("Client outdated (405) connect failure (client version: %s)", store.GetWAVersion().String())
		go cli.dispatchEvent(&events.ClientOutdated{})
	} else if reason == events.ConnectFailureCATInvalid || reason == events.ConnectFailureCATExpired {
		cli.Log.Infof("Got %d/%s connect failure, refreshing CAT before reconnecting...", int(reason), message)
		err := cli.RefreshCAT(ctx)
		if err != nil {
			cli.Log.Errorf("Failed to refresh CAT: %v", err)
			cli.expectDisconnect()
			go cli.dispatchEvent(&events.CATRefreshError{Error: err})
		}
	} else if willAutoReconnect {
		cli.Log.Warnf("Got %d/%s connect failure (location: %s), assuming automatic reconnect will handle it", int(reason), message, location)
	} else {
		cli.Log.Warnf("Unknown connect failure (location: %s): %s", location, node.XMLString())
		go cli.dispatchEvent(&events.ConnectFailure{Reason: reason, Message: message, Raw: node})
	}
}

func (cli *Client) handleConnectSuccess(ctx context.Context, node *waBinary.Node) {
	cli.Log.Infof("Successfully authenticated")
	cli.LastSuccessfulConnect = time.Now()
	cli.AutoReconnectErrors = 0
	cli.isLoggedIn.Store(true)
	nodeLID := node.AttrGetter().JID("lid")
	cli.Log.Debugf("[ConnectSuccess] Server provided LID in connect success node: %s (empty=%v)", nodeLID, nodeLID.IsEmpty())
	if !cli.Store.LID.IsEmpty() && !nodeLID.IsEmpty() && cli.Store.LID != nodeLID {
		// This should probably never happen, but check just in case.
		cli.Log.Warnf("[ConnectSuccess] Stored LID doesn't match one in connect success: %s != %s", cli.Store.LID, nodeLID)
		cli.Store.LID = types.EmptyJID
	}
	if cli.Store.LID.IsEmpty() && !nodeLID.IsEmpty() {
		cli.Store.LID = nodeLID
		err := cli.Store.Save(ctx)
		if err != nil {
			cli.Log.Warnf("[ConnectSuccess] Failed to save device after updating LID: %v", err)
		} else {
			cli.Log.Infof("[ConnectSuccess] Updated LID from server node: %s", cli.Store.LID)
		}
	} else if cli.Store.LID.IsEmpty() && nodeLID.IsEmpty() {
		cli.Log.Debugf("[ConnectSuccess] No LID provided in connect success node (nodeLID is empty)")
	}
	// If account is migrated but LID is still empty, query it proactively
	if cli.Store.LIDMigrationTimestamp > 0 && cli.Store.LID.IsEmpty() {
		ownJID := cli.Store.GetJID()
		if !ownJID.IsEmpty() {
			cli.Log.Infof("[ConnectSuccess] Account migrated (lid_migration_ts=%d) but own LID is empty, querying LID for %s using GetLIDOnly", cli.Store.LIDMigrationTimestamp, ownJID)
			lidMap, err := cli.GetLIDOnly(ctx, []types.JID{ownJID})
			if err == nil && !lidMap[ownJID].IsEmpty() {
				cli.Store.LID = lidMap[ownJID]
				err = cli.Store.Save(ctx)
				if err != nil {
					cli.Log.Warnf("[ConnectSuccess] Failed to save own LID after query: %v", err)
				} else {
					cli.Log.Infof("[ConnectSuccess] Successfully queried and saved own LID: %s (using GetLIDOnly query+message mode)", cli.Store.LID)
				}
			} else if err != nil {
				cli.Log.Warnf("[ConnectSuccess] GetLIDOnly failed to query own LID for %s: %v", ownJID, err)
			} else {
				cli.Log.Warnf("[ConnectSuccess] GetLIDOnly succeeded but own LID not found in server response for %s", ownJID)
			}
		} else {
			cli.Log.Warnf("[ConnectSuccess] Cannot query own LID: own JID is empty")
		}
	}
	// Some users are missing their own LID-PN mapping even though it's already in the device table,
	// so do this unconditionally for a few months to ensure everyone gets the row.
	cli.StoreLIDPNMapping(ctx, cli.Store.GetLID(), cli.Store.GetJID())
	
	// 连接成功后设置会话地理信息缓存（如果尚未设置）
	// 从数据库读取指纹并设置会话缓存（作为兜底，业务层应该已经通过 SetUserLoginGeoInfo 设置）
	if container, ok := cli.Store.Container.(*sqlstore.Container); ok {
		jid := cli.Store.GetJID()
		if !jid.IsEmpty() {
			// 检查会话缓存是否已设置
			sessionGeo := cli.getSessionGeoCache()
			if sessionGeo == nil || !sessionGeo.Locked {
				// 从数据库读取指纹
				if fp, err := container.GetFingerprint(ctx, jid); err == nil && fp != nil {
					if fp.LocaleCountry != "" && fp.LocaleLanguage != "" {
						// 从国家代码推断时区（兜底逻辑）
						timezone := getTimezoneByCountry(fp.LocaleCountry)
						cli.SetUserLoginGeoInfo(fp.LocaleCountry, timezone, fp.LocaleLanguage)
						if cli.Log != nil {
							cli.Log.Infof("[Fingerprint] Set session geo cache from database fingerprint: Country=%s, Timezone=%s, Language=%s",
								fp.LocaleCountry, timezone, fp.LocaleLanguage)
						}
					}
				}
			}
		}
	}
	
	go func() {
		// 添加超时控制，避免阻塞过久
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		
		if dbCount, err := cli.Store.PreKeys.UploadedPreKeyCount(ctx); err != nil {
			cli.Log.Errorf("Failed to get number of prekeys in database: %v", err)
		} else if serverCount, err := cli.getServerPreKeyCount(ctx); err != nil {
			cli.Log.Warnf("Failed to get number of prekeys on server: %v", err)
		} else {
			cli.Log.Debugf("Database has %d prekeys, server says we have %d", dbCount, serverCount)
			if serverCount < MinPreKeyCount || dbCount < MinPreKeyCount {
				cli.uploadPreKeys(ctx, dbCount == 0 && serverCount == 0)
				sc, _ := cli.getServerPreKeyCount(ctx)
				cli.Log.Debugf("Prekey count after upload: %d", sc)
			}
		}
		err := cli.SetPassive(ctx, false)
		if err != nil {
			cli.Log.Warnf("Failed to send post-connect passive IQ: %v", err)
		}
		cli.dispatchEvent(&events.Connected{})
		cli.closeSocketWaitChan()
	}()
}

// SetPassive tells the WhatsApp server whether this device is passive or not.
//
// This seems to mostly affect whether the device receives certain events.
// By default, whatsmeow will automatically do SetPassive(false) after connecting.
func (cli *Client) SetPassive(ctx context.Context, passive bool) error {
	tag := "active"
	if passive {
		tag = "passive"
	}
	_, err := cli.sendIQ(ctx, infoQuery{
		Namespace: "passive",
		Type:      "set",
		To:        types.ServerJID,
		Content:   []waBinary.Node{{Tag: tag}},
	})
	if err != nil {
		return err
	}
	return nil
}
