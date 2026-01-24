// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package whatsmeow implements a client for interacting with the WhatsApp web multidevice API.
package whatsmeow

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/exsync"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"golang.org/x/net/proxy"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/appstate"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waWa6"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/socket"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/util/fingerprint"
	"go.mau.fi/whatsmeow/util/keys"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// EventHandler is a function that can handle events from WhatsApp.
type EventHandler func(evt any)
type EventHandlerWithSuccessStatus func(evt any) bool
type nodeHandler func(ctx context.Context, node *waBinary.Node)

var nextHandlerID uint32

type wrappedEventHandler struct {
	fn EventHandlerWithSuccessStatus
	id uint32
}

type deviceCache struct {
	devices []types.JID
	dhash   string
}

// Client contains everything necessary to connect to and interact with the WhatsApp web API.
type Client struct {
	Store   *store.Device
	Log     waLog.Logger
	recvLog waLog.Logger
	sendLog waLog.Logger

	// EnablePNFallbackOnMissingLID controls whether SendMessage will fall back to PN send
	// when no LID is found for a PN even after querying the server.
	// Default is false, which keeps the original behavior.
	EnablePNFallbackOnMissingLID bool

	socket     *socket.NoiseSocket
	socketLock sync.RWMutex
	socketWait chan struct{}

	isLoggedIn            atomic.Bool
	expectedDisconnect    *exsync.Event
	forceAutoReconnect    atomic.Bool
	EnableAutoReconnect   bool
	InitialAutoReconnect  bool
	LastSuccessfulConnect time.Time
	AutoReconnectErrors   int
	// AutoReconnectHook is called when auto-reconnection fails. If the function returns false,
	// the client will not attempt to reconnect. The number of retries can be read from AutoReconnectErrors.
	AutoReconnectHook func(error) bool
	// If SynchronousAck is set, acks for messages will only be sent after all event handlers return.
	SynchronousAck             bool
	EnableDecryptedEventBuffer bool
	lastDecryptedBufferClear   time.Time

	DisableLoginAutoReconnect bool

	sendActiveReceipts atomic.Uint32

	// EmitAppStateEventsOnFullSync can be set to true if you want to get app state events emitted
	// even when re-syncing the whole state.
	EmitAppStateEventsOnFullSync bool
	AppStateDebugLogs            bool

	AutomaticMessageRerequestFromPhone bool
	pendingPhoneRerequests             map[types.MessageID]context.CancelFunc
	pendingPhoneRerequestsLock         sync.RWMutex

	appStateProc     *appstate.Processor
	appStateSyncLock sync.Mutex

	historySyncNotifications  chan *waE2E.HistorySyncNotification
	historySyncHandlerStarted atomic.Bool
	ManualHistorySyncDownload bool

	uploadPreKeysLock sync.Mutex
	lastPreKeyUpload  time.Time

	mediaConnCache *MediaConn
	mediaConnLock  sync.Mutex

	responseWaiters     map[string]chan<- *waBinary.Node
	responseWaitersLock sync.Mutex

	nodeHandlers      map[string]nodeHandler
	handlerQueue      chan *waBinary.Node
	eventHandlers     []wrappedEventHandler
	eventHandlersLock sync.RWMutex

	messageRetries     map[string]int
	messageRetriesLock sync.Mutex

	incomingRetryRequestCounter     map[incomingRetryKey]int
	incomingRetryRequestCounterLock sync.Mutex

	appStateKeyRequests     map[string]time.Time
	appStateKeyRequestsLock sync.RWMutex

	messageSendLock sync.Mutex

	privacySettingsCache atomic.Value

	groupCache               map[types.JID]*groupMetaCache
	groupCacheLock           sync.Mutex
	userDevicesCache         map[types.JID]deviceCache
	userDevicesCacheLock     sync.Mutex
	pendingDeviceQueries     map[types.JID]struct{}
	pendingDeviceQueriesLock sync.Mutex

	recentMessagesMap  map[recentMessageKey]RecentMessage
	recentMessagesList [recentMessagesSize]recentMessageKey
	recentMessagesPtr  int
	recentMessagesLock sync.RWMutex

	sessionRecreateHistory     map[types.JID]time.Time
	sessionRecreateHistoryLock sync.Mutex

	// pendingFingerprint 存储配对阶段生成的临时指纹，配对成功后保存到数据库
	pendingFingerprint      *store.DeviceFingerprint
	pendingFingerprintLock  sync.RWMutex // 使用 RWMutex 提高并发性能
	pendingFingerprintSaved atomic.Bool  // 标记是否已保存到数据库（避免重复保存）

	// GetMessageForRetry is used to find the source message for handling retry receipts
	// when the message is not found in the recently sent message cache.
	// Note: in DMs, the "to" field may be different from what you originally sent to (LID vs phone number),
	// make sure to check both if necessary.
	GetMessageForRetry func(requester, to types.JID, id types.MessageID) *waE2E.Message
	// PreRetryCallback is called before a retry receipt is accepted.
	// If it returns false, the accepting will be cancelled and the retry receipt will be ignored.
	PreRetryCallback func(receipt *events.Receipt, id types.MessageID, retryCount int, msg *waE2E.Message) bool

	// PrePairCallback is called before pairing is completed. If it returns false, the pairing will be cancelled and
	// the client will disconnect.
	PrePairCallback func(jid types.JID, platform, businessName string) bool

	// GetClientPayload is called to get the client payload for connecting to the server.
	// This should NOT be used for WhatsApp (to change the OS name, update fields in store.BaseClientPayload directly).
	GetClientPayload func() *waWa6.ClientPayload

	// Should untrusted identity errors be handled automatically? If true, the stored identity and existing signal
	// sessions will be removed on untrusted identity errors, and an events.IdentityChange will be dispatched.
	// If false, decrypting a message from untrusted devices will fail.
	AutoTrustIdentity bool

	// Should SubscribePresence return an error if no privacy token is stored for the user?
	ErrorOnSubscribePresenceWithoutToken bool

	SendReportingTokens bool

	BackgroundEventCtx context.Context

	phoneLinkingCache *phoneLinkingCache

	uniqueID  string
	idCounter atomic.Uint64

	mediaHTTP     *http.Client
	websocketHTTP *http.Client
	preLoginHTTP  *http.Client

	// This field changes the client to act like a Messenger client instead of a WhatsApp one.
	//
	// Note that you cannot use a Messenger account just by setting this field, you must use a
	// separate library for all the non-e2ee-related stuff like logging in.
	// The library is currently embedded in mautrix-meta (https://github.com/mautrix/meta), but may be separated later.
	MessengerConfig *MessengerConfig
	RefreshCAT      func(context.Context) error

	// CarrierInfo 外部传入的运营商信息（优先级最高）
	// 如果业务层通过前置探测获取到代理 IP 的运营商信息，应调用 SetCarrierInfo 设置
	carrierInfo     *CarrierInfo
	carrierInfoLock sync.RWMutex

	// sessionGeoCache 会话地理信息缓存（用于会话内锁定）
	// 首次设置后锁定，会话内禁止修改，断开连接时清除
	sessionGeoCache *SessionGeoCache
	sessionGeoLock  sync.RWMutex
}

type groupMetaCache struct {
	AddressingMode             types.AddressingMode
	CommunityAnnouncementGroup bool
	Members                    []types.JID
}

type MessengerConfig struct {
	UserAgent    string
	BaseURL      string
	WebsocketURL string
}

// CarrierInfo 运营商信息（从代理 IP 探测获取）
type CarrierInfo struct {
	MCC string // 移动国家代码 (如 "404", "405", "724")
	MNC string // 移动网络代码（如 "000" 表示固定宽带，或其他值表示移动网络）
}

// SessionGeoCache 会话地理信息缓存
// 用于会话内锁定 Country/Timezone/Language，保证最小封控率
type SessionGeoCache struct {
	Country  string // 国家代码，如 "IN", "BR"
	Timezone string // 时区，如 "Asia/Kolkata"（仅用于锁定，不写入Payload）
	Language string // 语言代码，如 "hi", "pt", "en"
	Locked   bool   // 是否已锁定
}

// Size of buffer for the channel that all incoming XML nodes go through.
// In general it shouldn't go past a few buffered messages, but the channel is big to be safe.
const handlerQueueSize = 2048

// NewClient initializes a new WhatsApp web client.
//
// The logger can be nil, it will default to a no-op logger.
//
// The device store must be set. A default SQL-backed implementation is available in the store/sqlstore package.
//
//	container, err := sqlstore.New("sqlite3", "file:yoursqlitefile.db?_foreign_keys=on", nil)
//	if err != nil {
//		panic(err)
//	}
//	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
//	deviceStore, err := container.GetFirstDevice()
//	if err != nil {
//		panic(err)
//	}
//	client := whatsmeow.NewClient(deviceStore, nil)
func NewClient(deviceStore *store.Device, log waLog.Logger) *Client {
	if log == nil {
		log = waLog.Noop
	}
	uniqueIDPrefix := random.Bytes(2)
	baseHTTPClient := &http.Client{
		Transport: (http.DefaultTransport.(*http.Transport)).Clone(),
	}
	cli := &Client{
		mediaHTTP:          ptr.Clone(baseHTTPClient),
		websocketHTTP:      ptr.Clone(baseHTTPClient),
		preLoginHTTP:       ptr.Clone(baseHTTPClient),
		Store:              deviceStore,
		Log:                log,
		recvLog:            log.Sub("Recv"),
		sendLog:            log.Sub("Send"),
		uniqueID:           fmt.Sprintf("%d.%d-", uniqueIDPrefix[0], uniqueIDPrefix[1]),
		responseWaiters:    make(map[string]chan<- *waBinary.Node),
		eventHandlers:      make([]wrappedEventHandler, 0, 1),
		messageRetries:     make(map[string]int),
		handlerQueue:       make(chan *waBinary.Node, handlerQueueSize),
		appStateProc:       appstate.NewProcessor(deviceStore, log.Sub("AppState")),
		socketWait:         make(chan struct{}),
		expectedDisconnect: exsync.NewEvent(),

		incomingRetryRequestCounter: make(map[incomingRetryKey]int),

		historySyncNotifications: make(chan *waE2E.HistorySyncNotification, 32),

		groupCache:           make(map[types.JID]*groupMetaCache),
		userDevicesCache:     make(map[types.JID]deviceCache),
		pendingDeviceQueries: make(map[types.JID]struct{}),

		recentMessagesMap:      make(map[recentMessageKey]RecentMessage, recentMessagesSize),
		sessionRecreateHistory: make(map[types.JID]time.Time),
		GetMessageForRetry:     func(requester, to types.JID, id types.MessageID) *waE2E.Message { return nil },
		appStateKeyRequests:    make(map[string]time.Time),

		pendingPhoneRerequests: make(map[types.MessageID]context.CancelFunc),

		EnableAutoReconnect: true,
		AutoTrustIdentity:   true,

		BackgroundEventCtx: context.Background(),
	}
	cli.nodeHandlers = map[string]nodeHandler{
		"message":      cli.handleEncryptedMessage,
		"appdata":      cli.handleEncryptedMessage,
		"receipt":      cli.handleReceipt,
		"call":         cli.handleCallEvent,
		"chatstate":    cli.handleChatState,
		"presence":     cli.handlePresence,
		"notification": cli.handleNotification,
		"success":      cli.handleConnectSuccess,
		"failure":      cli.handleConnectFailure,
		"stream:error": cli.handleStreamError,
		"iq":           cli.handleIQ,
		"ib":           cli.handleIB,
		// Apparently there's also an <error> node which can have a code=479 and means "Invalid stanza sent (smax-invalid)"
	}

	// Automatically setup fingerprint callback if Container has fingerprint configuration
	setupFingerprintIfEnabled(cli, deviceStore)

	return cli
}

// setupFingerprintIfEnabled automatically sets up the fingerprint callback if the Container
// has fingerprint configuration enabled. This is called automatically by NewClient.
func setupFingerprintIfEnabled(cli *Client, deviceStore *store.Device) {
	// Check if Container is sqlstore.Container
	container, ok := deviceStore.Container.(*sqlstore.Container)
	if !ok {
		return
	}

	// If no region is configured, fingerprint is disabled
	if !container.FingerprintRegion.IsValid() {
		return
	}

	// Convert enum to string for GenerateFingerprint
	regionCode := container.FingerprintRegion.String()

	// Setup the callback (fully automated, no business layer intervention needed)
	cli.GetClientPayload = func() *waWa6.ClientPayload {
		payload := deviceStore.GetClientPayload()
		jid := deviceStore.GetJID()

		// 1. 获取会话锁定的地理信息（如果已锁定）
		sessionGeo := cli.getSessionGeoCache()

		// 2. 获取业务层传入的 MCC/MNC（如果已设置）
		carrierInfo := cli.GetCarrierInfo()

		ctx := context.Background()
		var fp *store.DeviceFingerprint
		var err error

		// regionCode 仅用于设备指纹生成（设备型号、语言分布等），不再用于地理信息
		regionCodeForDevice := regionCode

		if jid == types.EmptyJID {
			// 配对阶段：尝试从数据库读取临时指纹（使用电话号码 JID）
			var tempJID types.JID
			if cli.phoneLinkingCache != nil && !cli.phoneLinkingCache.jid.IsEmpty() {
				// 使用配对码的电话号码 JID
				tempJID = cli.phoneLinkingCache.jid
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] Pairing phase: attempting to load temporary fingerprint for phone %s", tempJID.User)
				}
			} else {
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] Pairing phase: QR code pairing (no phone number), using in-memory fingerprint")
				}
			}

			if !tempJID.IsEmpty() {
				// 尝试从数据库读取之前保存的临时指纹
				fp, err = container.GetFingerprint(ctx, tempJID)
				if err != nil {
					cli.Log.Warnf("[Fingerprint] Failed to get temporary fingerprint for %s: %v", tempJID.User, err)
				} else if fp != nil {
					if cli.Log != nil {
						cli.Log.Infof("[Fingerprint] Found temporary fingerprint in database for %s", tempJID.User)
					}
				} else {
					if cli.Log != nil {
						cli.Log.Infof("[Fingerprint] No temporary fingerprint found in database for %s, will generate new one", tempJID.User)
					}
				}
			}

			// 如果数据库中没有，生成新的临时指纹
			if fp == nil {
				cli.pendingFingerprintLock.Lock()
				if cli.pendingFingerprint == nil {
					// 生成临时指纹（仅用于设备特征，地理信息由业务层传入）
					cli.pendingFingerprint = fingerprint.GenerateFingerprint(regionCodeForDevice)
					// 应用业务层传入的 MCC/MNC（如果已设置，允许只设置其中一个）
					if carrierInfo != nil {
						if carrierInfo.MCC != "" {
							cli.pendingFingerprint.Mcc = carrierInfo.MCC
						}
						if carrierInfo.MNC != "" {
							cli.pendingFingerprint.Mnc = carrierInfo.MNC
						}
					}
					// 应用会话锁定的地理信息（如果已锁定）
					if sessionGeo != nil && sessionGeo.Locked {
						cli.pendingFingerprint.LocaleCountry = sessionGeo.Country
						cli.pendingFingerprint.LocaleLanguage = sessionGeo.Language
					}
					cli.pendingFingerprintSaved.Store(false) // 重置保存标记
					if cli.Log != nil {
						cli.Log.Infof("Generated temporary fingerprint for pairing: %s %s (%s %s), MCC: %s, MNC: %s, Country: %s, Language: %s",
							cli.pendingFingerprint.Manufacturer, cli.pendingFingerprint.Device,
							cli.pendingFingerprint.DevicePropsOs, cli.pendingFingerprint.OsVersion,
							cli.pendingFingerprint.Mcc, cli.pendingFingerprint.Mnc,
							cli.pendingFingerprint.LocaleCountry, cli.pendingFingerprint.LocaleLanguage)
					}
				}
				fp = cli.pendingFingerprint
				cli.pendingFingerprintLock.Unlock()

				// 如果有电话号码 JID，保存临时指纹到数据库（供下次复用）
				// 使用原子操作避免并发重复保存
				if !tempJID.IsEmpty() {
					if cli.pendingFingerprintSaved.CompareAndSwap(false, true) {
						if cli.Log != nil {
							cli.Log.Infof("[Fingerprint] Saving temporary fingerprint to database for %s (atomic operation)", tempJID.User)
						}
						// 复制指纹结构体，避免并发修改
						fpCopy := *fp
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()
							if saveErr := container.PutFingerprint(ctx, tempJID, &fpCopy); saveErr != nil {
								cli.Log.Warnf("[Fingerprint] Failed to save temporary fingerprint for %s: %v", tempJID.User, saveErr)
								cli.pendingFingerprintSaved.Store(false) // 保存失败，重置标记
							} else {
								cli.Log.Infof("[Fingerprint] Successfully saved temporary fingerprint for %s (will be reused if pairing times out)", tempJID.User)
							}
						}()
					} else {
						if cli.Log != nil {
							cli.Log.Debugf("[Fingerprint] Skipping duplicate save for %s (already saved by another goroutine)", tempJID.User)
						}
					}
				}
			} else {
				// 从数据库读取到临时指纹，记录日志并设置到内存中（供配对成功后使用）
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] Reusing temporary fingerprint for %s from database (fingerprint: %s %s, MCC: %s, MNC: %s)",
						tempJID.User, fp.Manufacturer, fp.Device, fp.Mcc, fp.Mnc)
				}
				// 设置到内存中，供配对成功后使用
				cli.pendingFingerprintLock.Lock()
				cli.pendingFingerprint = fp
				cli.pendingFingerprintSaved.Store(true) // 已从数据库读取，标记为已保存
				cli.pendingFingerprintLock.Unlock()
			}
		} else {
			// 已配对：从数据库读取或使用临时指纹
			if cli.Log != nil {
				cli.Log.Debugf("[Fingerprint] Paired device: loading fingerprint for JID %s", jid.User)
			}
			fp, err = container.GetFingerprint(ctx, jid)
			if err != nil {
				cli.Log.Warnf("[Fingerprint] Failed to get fingerprint for %s: %v", jid.User, err)
				// 尝试使用临时指纹
				cli.pendingFingerprintLock.Lock()
				if cli.pendingFingerprint != nil {
					fp = cli.pendingFingerprint
					cli.pendingFingerprintLock.Unlock()
					if cli.Log != nil {
						cli.Log.Infof("[Fingerprint] Using pending fingerprint for %s (database read failed)", jid.User)
					}
					// 复制指纹结构体，避免并发修改
					fpCopy := *fp
					// 保存临时指纹到数据库（使用原子操作避免重复保存）
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()
						if saveErr := container.PutFingerprint(ctx, jid, &fpCopy); saveErr != nil {
							cli.Log.Warnf("[Fingerprint] Failed to save pending fingerprint for %s: %v", jid.User, saveErr)
						} else {
							cli.Log.Infof("[Fingerprint] Saved pending fingerprint for %s (recovered from memory)", jid.User)
							// 清除临时指纹
							cli.pendingFingerprintLock.Lock()
							cli.pendingFingerprint = nil
							cli.pendingFingerprintSaved.Store(false)
							cli.pendingFingerprintLock.Unlock()
						}
					}()
				} else {
					cli.pendingFingerprintLock.Unlock()
					if cli.Log != nil {
						cli.Log.Warnf("[Fingerprint] No fingerprint found for %s and no pending fingerprint available", jid.User)
					}
				}
			} else if fp == nil {
				// 数据库中没有指纹，检查是否有临时指纹
				if cli.Log != nil {
					cli.Log.Infof("[Fingerprint] No fingerprint in database for %s, checking pending fingerprint", jid.User)
				}
				cli.pendingFingerprintLock.Lock()
				if cli.pendingFingerprint != nil {
					fp = cli.pendingFingerprint
					cli.pendingFingerprintLock.Unlock()
					if cli.Log != nil {
						cli.Log.Infof("[Fingerprint] Using pending fingerprint for %s (migrating to database)", jid.User)
					}
					// 复制指纹结构体，避免并发修改
					fpCopy := *fp
					// 保存临时指纹到数据库（使用原子操作避免重复保存）
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()
						if saveErr := container.PutFingerprint(ctx, jid, &fpCopy); saveErr != nil {
							cli.Log.Warnf("[Fingerprint] Failed to save pending fingerprint for %s: %v", jid.User, saveErr)
						} else {
							cli.Log.Infof("[Fingerprint] Successfully migrated pending fingerprint for %s to database", jid.User)
							// 清除临时指纹
							cli.pendingFingerprintLock.Lock()
							cli.pendingFingerprint = nil
							cli.pendingFingerprintSaved.Store(false)
							cli.pendingFingerprintLock.Unlock()
						}
					}()
				} else {
					cli.pendingFingerprintLock.Unlock()
					// 多次匹配（再次配对）：尝试复用数据库中的主指纹（如果存在）
					// 按照方案要求，同一账号的设备特征永远相同
					existingFp, getErr := container.GetFingerprint(ctx, jid)
					if getErr == nil && existingFp != nil {
						// 复用主指纹（设备特征完全不变）
						fp = existingFp
						if cli.Log != nil {
							cli.Log.Infof("[Fingerprint] Reusing main fingerprint for re-pairing %s: %s %s (%s %s), MCC: %s, MNC: %s (device features unchanged)",
								jid.User, fp.Manufacturer, fp.Device, fp.DevicePropsOs, fp.OsVersion, fp.Mcc, fp.Mnc)
						}
						// 仅更新地理信息和运营商信息（允许变更字段）
						if carrierInfo != nil {
							if carrierInfo.MCC != "" {
								fp.Mcc = carrierInfo.MCC
							}
							if carrierInfo.MNC != "" {
								fp.Mnc = carrierInfo.MNC
							}
						}
						if sessionGeo != nil && sessionGeo.Locked {
							fp.LocaleCountry = sessionGeo.Country
							fp.LocaleLanguage = sessionGeo.Language
						}
						// 更新数据库中的指纹（仅更新地理信息和运营商信息）
						fpCopy := *fp
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()
							if saveErr := container.PutFingerprint(ctx, jid, &fpCopy); saveErr != nil {
								cli.Log.Warnf("[Fingerprint] Failed to update fingerprint for %s: %v", jid.User, saveErr)
							} else {
								cli.Log.Infof("[Fingerprint] Successfully updated fingerprint for %s (geo/carrier only)", jid.User)
							}
						}()
					} else {
						// 数据库中没有主指纹，生成新指纹（首次匹配）
						fp = fingerprint.GenerateFingerprint(regionCodeForDevice)
						// 应用业务层传入的 MCC/MNC（如果已设置，允许只设置其中一个）
						if carrierInfo != nil {
							if carrierInfo.MCC != "" {
								fp.Mcc = carrierInfo.MCC
							}
							if carrierInfo.MNC != "" {
								fp.Mnc = carrierInfo.MNC
							}
						}
						// 应用会话锁定的地理信息（如果已锁定）
						if sessionGeo != nil && sessionGeo.Locked {
							fp.LocaleCountry = sessionGeo.Country
							fp.LocaleLanguage = sessionGeo.Language
						}
						if cli.Log != nil {
							cli.Log.Infof("[Fingerprint] Generated new fingerprint for %s: %s %s (%s %s), MCC: %s, MNC: %s, Country: %s, Language: %s",
								jid.User, fp.Manufacturer, fp.Device, fp.DevicePropsOs, fp.OsVersion,
								fp.Mcc, fp.Mnc, fp.LocaleCountry, fp.LocaleLanguage)
						}
						// 复制指纹结构体，避免并发修改
						fpCopy := *fp
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()
							if saveErr := container.PutFingerprint(ctx, jid, &fpCopy); saveErr != nil {
								cli.Log.Warnf("[Fingerprint] Failed to save new fingerprint for %s: %v", jid.User, saveErr)
							} else {
								cli.Log.Infof("[Fingerprint] Successfully saved new fingerprint for %s", jid.User)
							}
						}()
					}
				}
			} else {
				// 成功从数据库读取指纹，清除临时指纹（如果存在）
				if cli.Log != nil {
					cli.Log.Debugf("[Fingerprint] Loaded fingerprint from database for %s (fingerprint: %s %s, MCC: %s, MNC: %s)",
						jid.User, fp.Manufacturer, fp.Device, fp.Mcc, fp.Mnc)
				}
				cli.pendingFingerprintLock.Lock()
				hadPending := cli.pendingFingerprint != nil
				cli.pendingFingerprint = nil
				cli.pendingFingerprintSaved.Store(false)
				cli.pendingFingerprintLock.Unlock()
				if hadPending && cli.Log != nil {
					cli.Log.Debugf("[Fingerprint] Cleared pending fingerprint for %s (using database fingerprint)", jid.User)
				}
			}
		}

		if fp != nil {
			// 1. 应用业务层传入的 MCC/MNC（如果已设置，允许只设置其中一个）
			if carrierInfo != nil {
				if carrierInfo.MCC != "" {
					if fp.Mcc != carrierInfo.MCC {
						if cli.Log != nil {
							cli.Log.Debugf("[Fingerprint] Applying carrier MCC: %s -> %s", fp.Mcc, carrierInfo.MCC)
						}
						fp.Mcc = carrierInfo.MCC
					}
				}
				if carrierInfo.MNC != "" {
					if fp.Mnc != carrierInfo.MNC {
						if cli.Log != nil {
							cli.Log.Debugf("[Fingerprint] Applying carrier MNC: %s -> %s", fp.Mnc, carrierInfo.MNC)
						}
						fp.Mnc = carrierInfo.MNC
					}
				}
			}

			// 2. 应用会话锁定的地理信息（如果已锁定，强制使用，禁止修改）
			if sessionGeo != nil && sessionGeo.Locked {
				if fp.LocaleCountry != sessionGeo.Country || fp.LocaleLanguage != sessionGeo.Language {
					if cli.Log != nil {
						cli.Log.Debugf("[Fingerprint] Applying session locked geo info: Country=%s->%s, Language=%s->%s",
							fp.LocaleCountry, sessionGeo.Country, fp.LocaleLanguage, sessionGeo.Language)
					}
					fp.LocaleCountry = sessionGeo.Country
					fp.LocaleLanguage = sessionGeo.Language
				}
			}

			// 3. 应用指纹到 Payload（regionCode 仅用于设备指纹生成）
			fingerprint.ApplyFingerprint(payload, fp, regionCodeForDevice)

			// 4. 最终清理：确保字段完整性（流程闭环）
			payload = store.SanitizeClientPayload(payload)
		}

		return payload
	}
}

// SetCarrierInfo 设置外部传入的运营商信息（从代理 IP 探测获取）
// 业务层通过前置探测获取到代理 IP 的运营商信息后，应调用此方法设置
// 参数：
//   - mcc: 移动国家代码 (如 "404", "405", "724")，如果为空字符串，将清除已设置的 MCC
//   - mnc: 移动网络代码，如果为空字符串，将清除已设置的 MNC（如果都为空，将在应用指纹时按优先级设置）
func (cli *Client) SetCarrierInfo(mcc, mnc string) {
	cli.carrierInfoLock.Lock()
	defer cli.carrierInfoLock.Unlock()

	// 检查是否有变更
	hasChange := false
	if cli.carrierInfo == nil {
		hasChange = (mcc != "" || mnc != "")
	} else {
		hasChange = (cli.carrierInfo.MCC != mcc || cli.carrierInfo.MNC != mnc)
	}

	// 允许只设置 MCC 或只设置 MNC
	if mcc != "" || mnc != "" {
		if cli.carrierInfo == nil {
			cli.carrierInfo = &CarrierInfo{}
		}
		if mcc != "" {
			cli.carrierInfo.MCC = mcc
		} else {
			cli.carrierInfo.MCC = "" // 如果传入空字符串，清除 MCC
		}
		if mnc != "" {
			cli.carrierInfo.MNC = mnc
		} else {
			cli.carrierInfo.MNC = "" // 如果传入空字符串，清除 MNC
		}
		if cli.Log != nil {
			cli.Log.Infof("[Fingerprint] Set carrier info from proxy IP: MCC=%s, MNC=%s", cli.carrierInfo.MCC, cli.carrierInfo.MNC)
			if hasChange && cli.socket != nil {
				cli.Log.Warnf("[Fingerprint] Carrier info changed while connected (MCC=%s, MNC=%s). Consider reconnecting to apply new MCC/MNC", cli.carrierInfo.MCC, cli.carrierInfo.MNC)
			}
		}
	} else {
		// 如果 MCC 和 MNC 都为空，清除 carrierInfo
		cli.carrierInfo = nil
		if cli.Log != nil {
			cli.Log.Debugf("[Fingerprint] Cleared carrier info (empty MCC/MNC provided)")
		}
	}
}

// GetCarrierInfo 获取当前设置的运营商信息（线程安全）
// 返回值的副本，避免并发修改问题
func (cli *Client) GetCarrierInfo() *CarrierInfo {
	cli.carrierInfoLock.RLock()
	defer cli.carrierInfoLock.RUnlock()
	if cli.carrierInfo == nil {
		return nil
	}
	// 返回副本，避免外部修改导致并发问题
	return &CarrierInfo{
		MCC: cli.carrierInfo.MCC,
		MNC: cli.carrierInfo.MNC,
	}
}

// getTimezoneByCountry 根据国家代码推断时区（兜底逻辑）
// 业务层应该通过 SetUserLoginGeoInfo 传入准确的时区
func getTimezoneByCountry(country string) string {
	timezoneMap := map[string]string{
		"IN": "Asia/Kolkata",         // 印度
		"BR": "America/Sao_Paulo",    // 巴西
		"US": "America/New_York",     // 美国
		"GB": "Europe/London",        // 英国
		"CA": "America/Toronto",      // 加拿大
		"AU": "Australia/Sydney",     // 澳大利亚
		"NZ": "Pacific/Auckland",     // 新西兰
		"CN": "Asia/Shanghai",        // 中国
		"TW": "Asia/Taipei",          // 台湾
		"HK": "Asia/Hong_Kong",       // 香港
		"SG": "Asia/Singapore",       // 新加坡
		"ES": "Europe/Madrid",        // 西班牙
		"MX": "America/Mexico_City",  // 墨西哥
		"AR": "America/Buenos_Aires", // 阿根廷
		"CO": "America/Bogota",       // 哥伦比亚
		"FR": "Europe/Paris",         // 法国
		"DE": "Europe/Berlin",        // 德国
		"AT": "Europe/Vienna",        // 奥地利
		"JP": "Asia/Tokyo",           // 日本
		"KR": "Asia/Seoul",           // 韩国
	}
	if tz, ok := timezoneMap[country]; ok {
		return tz
	}
	// 默认返回 UTC（如果国家代码未匹配）
	return "UTC"
}

// SetUserLoginGeoInfo 设置用户登录Web端IP的地理信息（每次匹配时传入）
// 业务层在每次匹配/登录时调用此方法传入用户真实地理位置
// 参数：
//   - country: 国家代码，如 "IN", "BR"
//   - timezone: 时区，如 "Asia/Kolkata"（仅用于会话锁定，不写入Payload）
//   - language: 语言代码，如 "hi", "pt", "en"
//
// 首次设置后会话内锁定，禁止修改；断开连接时清除锁定；重登时允许重新设置
func (cli *Client) SetUserLoginGeoInfo(country, timezone, language string) {
	if country == "" {
		if cli.Log != nil {
			cli.Log.Warnf("[Fingerprint] SetUserLoginGeoInfo: country is empty, ignoring")
		}
		return
	}

	cli.sessionGeoLock.Lock()
	defer cli.sessionGeoLock.Unlock()

	cli.sessionGeoCache = &SessionGeoCache{
		Country:  country,
		Timezone: timezone,
		Language: language,
		Locked:   true,
	}

	if cli.Log != nil {
		cli.Log.Infof("[Fingerprint] Set user login geo info: Country=%s, Timezone=%s, Language=%s (locked)", country, timezone, language)
	}
}

// getSessionGeoCache 获取会话地理信息缓存（线程安全）
func (cli *Client) getSessionGeoCache() *SessionGeoCache {
	cli.sessionGeoLock.RLock()
	defer cli.sessionGeoLock.RUnlock()
	if cli.sessionGeoCache == nil || !cli.sessionGeoCache.Locked {
		return nil
	}
	// 返回副本，避免外部修改
	return &SessionGeoCache{
		Country:  cli.sessionGeoCache.Country,
		Timezone: cli.sessionGeoCache.Timezone,
		Language: cli.sessionGeoCache.Language,
		Locked:   cli.sessionGeoCache.Locked,
	}
}

// clearSessionGeoCache 清除会话地理信息缓存（断开连接时调用）
func (cli *Client) clearSessionGeoCache() {
	cli.sessionGeoLock.Lock()
	defer cli.sessionGeoLock.Unlock()
	if cli.sessionGeoCache != nil {
		if cli.Log != nil {
			cli.Log.Debugf("[Fingerprint] Cleared session geo cache: Country=%s, Timezone=%s, Language=%s", cli.sessionGeoCache.Country, cli.sessionGeoCache.Timezone, cli.sessionGeoCache.Language)
		}
		cli.sessionGeoCache = nil
	}
}

// validateClientPayload 校验 Payload 是否符合方案要求
// 返回错误列表（如果有不一致的字段）
func (cli *Client) validateClientPayload(payload *waWa6.ClientPayload) error {
	if payload == nil {
		return fmt.Errorf("payload is nil")
	}

	var issues []string
	ua := payload.UserAgent
	if ua == nil {
		return fmt.Errorf("UserAgent is nil")
	}

	// 1. 地理一致性校验
	country := ua.GetLocaleCountryIso31661Alpha2()
	language := ua.GetLocaleLanguageIso6391()
	if country != "" && language != "" {
		// 检查语言与国家匹配（如 "IN" ↔ "hi", "BR" ↔ "pt"）
		if !isLanguageCountryMatch(language, country) {
			issues = append(issues, fmt.Sprintf("Language-Country mismatch: %s-%s", language, country))
		}
	}

	// 2. 网络一致性校验
	if ua.GetMnc() != "000" {
		issues = append(issues, fmt.Sprintf("MNC=%s (should be 000)", ua.GetMnc()))
	}
	if country != "" && ua.GetMcc() != "" {
		// 检查 MCC 与 Country 匹配（如 "404" ↔ "IN", "724" ↔ "BR"）
		if !isMCCCountryMatch(ua.GetMcc(), country) {
			issues = append(issues, fmt.Sprintf("MCC-Country mismatch: MCC=%s, Country=%s", ua.GetMcc(), country))
		}
	}

	// 3. 平台一致性校验
	if ua.GetPlatform() != waWa6.ClientPayload_UserAgent_WEB {
		issues = append(issues, fmt.Sprintf("Platform=%v (should be WEB)", ua.GetPlatform()))
	}
	if ua.GetDeviceType() != waWa6.ClientPayload_UserAgent_DESKTOP {
		issues = append(issues, fmt.Sprintf("DeviceType=%v (should be DESKTOP)", ua.GetDeviceType()))
	}
	if payload.WebInfo == nil || payload.WebInfo.GetWebSubPlatform() != waWa6.ClientPayload_WebInfo_WEB_BROWSER {
		issues = append(issues, "WebInfo.WebSubPlatform should be WEB_BROWSER")
	}

	// 4. WEB 平台特征校验
	if ua.GetPlatform() == waWa6.ClientPayload_UserAgent_WEB {
		if ua.OsBuildNumber != nil {
			issues = append(issues, "OsBuildNumber should be nil for WEB platform")
		}
		if ua.DeviceBoard != nil {
			issues = append(issues, "DeviceBoard should be nil for WEB platform")
		}
		if ua.DeviceModelType != nil {
			issues = append(issues, "DeviceModelType should be nil for WEB platform")
		}
		if ua.GetDevice() != "Desktop" {
			issues = append(issues, fmt.Sprintf("Device=%s (should be Desktop for WEB)", ua.GetDevice()))
		}
		if ua.PhoneID != nil {
			issues = append(issues, "PhoneID should be nil for WEB platform")
		}
		if ua.DeviceExpID != nil {
			issues = append(issues, "DeviceExpID should be nil for WEB platform")
		}
	}

	// 5. 会话锁定校验
	sessionGeo := cli.getSessionGeoCache()
	if sessionGeo != nil && sessionGeo.Locked {
		if ua.GetLocaleCountryIso31661Alpha2() != sessionGeo.Country {
			issues = append(issues, fmt.Sprintf("Country mismatch with session cache: %s != %s", ua.GetLocaleCountryIso31661Alpha2(), sessionGeo.Country))
		}
		if ua.GetLocaleLanguageIso6391() != sessionGeo.Language {
			issues = append(issues, fmt.Sprintf("Language mismatch with session cache: %s != %s", ua.GetLocaleLanguageIso6391(), sessionGeo.Language))
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("validation failed: %v", issues)
	}
	return nil
}

// fixClientPayload 修正 Payload 中的不一致字段（使用兜底值）
func (cli *Client) fixClientPayload(payload *waWa6.ClientPayload) *waWa6.ClientPayload {
	if payload == nil {
		return nil
	}

	ua := payload.UserAgent
	if ua == nil {
		return payload
	}

	// 注意：不强制修正 MNC，因为外界可能设置移动宽带的 MNC
	// MNC 的优先级已在 ApplyFingerprint 中处理：外界 > 地区配置 > "000" > 空

	// 强制修正 WEB 平台特征
	if ua.GetPlatform() == waWa6.ClientPayload_UserAgent_WEB {
		ua.OsBuildNumber = nil
		ua.DeviceBoard = nil
		ua.DeviceModelType = nil
		ua.Device = proto.String("Desktop")
		ua.PhoneID = nil
		ua.DeviceExpID = nil
	}

	// 强制修正会话锁定的地理信息
	sessionGeo := cli.getSessionGeoCache()
	if sessionGeo != nil && sessionGeo.Locked {
		ua.LocaleCountryIso31661Alpha2 = proto.String(sessionGeo.Country)
		ua.LocaleLanguageIso6391 = proto.String(sessionGeo.Language)

		// 修正 MCC 以匹配 Country（如果不匹配）
		if ua.GetMcc() != "" && !isMCCCountryMatch(ua.GetMcc(), sessionGeo.Country) {
			defaultMCC := fingerprint.GetDefaultMCCByCountry(sessionGeo.Country)
			if cli.Log != nil {
				cli.Log.Warnf("[Fingerprint] MCC-Country mismatch detected (MCC=%s, Country=%s), fixing to MCC=%s",
					ua.GetMcc(), sessionGeo.Country, defaultMCC)
			}
			ua.Mcc = proto.String(defaultMCC)
		}
	}

	// 强制修正 WebInfo
	if payload.WebInfo == nil {
		payload.WebInfo = &waWa6.ClientPayload_WebInfo{
			WebSubPlatform: waWa6.ClientPayload_WebInfo_WEB_BROWSER.Enum(),
		}
	} else if payload.WebInfo.WebSubPlatform == nil {
		payload.WebInfo.WebSubPlatform = waWa6.ClientPayload_WebInfo_WEB_BROWSER.Enum()
	}

	return payload
}

// isLanguageCountryMatch 检查语言与国家是否匹配
func isLanguageCountryMatch(language, country string) bool {
	matches := map[string][]string{
		"hi": {"IN"},
		"pt": {"BR"},
		"en": {"US", "GB", "CA", "AU", "NZ"},
		"es": {"ES", "MX", "AR", "CO"},
		"fr": {"FR"},
		"de": {"DE", "AT"},
		"ja": {"JP"},
		"ko": {"KR"},
		"zh": {"CN", "TW", "HK", "SG"},
	}
	if countries, ok := matches[language]; ok {
		for _, c := range countries {
			if c == country {
				return true
			}
		}
	}
	// 如果未匹配到，允许（可能是其他语言）
	return true
}

// isMCCCountryMatch 检查 MCC 与国家是否匹配
func isMCCCountryMatch(mcc, country string) bool {
	mccMap := map[string]string{
		// 印度 (India) - 2个MCC
		"404": "IN", // 印度 GSM/UMTS/LTE (Airtel, Vodafone, BSNL等)
		"405": "IN", // 印度 LTE/CDMA (Reliance Jio等)

		// 巴西 (Brazil) - 1个MCC
		"724": "BR", // 巴西 (TIM, Vivo, Claro, Oi等)

		// 其他国家
		"310": "US", // 美国
		"234": "GB", // 英国
		"302": "CA", // 加拿大
		"505": "AU", // 澳大利亚
		"530": "NZ", // 新西兰
		"460": "CN", // 中国
		"466": "TW", // 台湾
		"454": "HK", // 香港
		"525": "SG", // 新加坡
		"214": "ES", // 西班牙
		"334": "MX", // 墨西哥
		"722": "AR", // 阿根廷
		"732": "CO", // 哥伦比亚
		"208": "FR", // 法国
		"206": "BE", // 比利时
		"228": "CH", // 瑞士
		"262": "DE", // 德国
		"232": "AT", // 奥地利
		"440": "JP", // 日本
		"450": "KR", // 韩国
	}
	if expectedCountry, ok := mccMap[mcc]; ok {
		return expectedCountry == country
	}
	// 如果未匹配到，允许（可能是其他MCC）
	return true
}

// SetProxyAddress is a helper method that parses a URL string and calls SetProxy or SetSOCKSProxy based on the URL scheme.
//
// Returns an error if url.Parse fails to parse the given address.
func (cli *Client) SetProxyAddress(addr string, opts ...SetProxyOptions) error {
	if addr == "" {
		cli.SetProxy(nil, opts...)
		return nil
	}
	parsed, err := url.Parse(addr)
	if err != nil {
		return err
	}
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		cli.SetProxy(http.ProxyURL(parsed), opts...)
	} else if parsed.Scheme == "socks5" {
		px, err := proxy.FromURL(parsed, &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		})
		if err != nil {
			return err
		}
		cli.SetSOCKSProxy(px, opts...)
	} else {
		return fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}
	return nil
}

type Proxy = func(*http.Request) (*url.URL, error)

// SetProxy sets a HTTP proxy to use for WhatsApp web websocket connections and media uploads/downloads.
//
// Must be called before Connect() to take effect in the websocket connection.
// If you want to change the proxy after connecting, you must call Disconnect() and then Connect() again manually.
//
// By default, the client will find the proxy from the https_proxy environment variable like Go's net/http does.
//
// To disable reading proxy info from environment variables, explicitly set the proxy to nil:
//
//	cli.SetProxy(nil)
//
// To use a different proxy for the websocket and media, pass a function that checks the request path or headers:
//
//	cli.SetProxy(func(r *http.Request) (*url.URL, error) {
//		if r.URL.Host == "web.whatsapp.com" && r.URL.Path == "/ws/chat" {
//			return websocketProxyURL, nil
//		} else {
//			return mediaProxyURL, nil
//		}
//	})
func (cli *Client) SetProxy(proxy Proxy, opts ...SetProxyOptions) {
	var opt SetProxyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	transport := (http.DefaultTransport.(*http.Transport)).Clone()
	transport.Proxy = proxy
	cli.setTransport(transport, opt)
}

type SetProxyOptions struct {
	// If NoWebsocket is true, the proxy won't be used for the websocket
	NoWebsocket bool
	// If OnlyLogin is true, the proxy will be used for the pre-login websocket, but not the post-login one
	OnlyLogin bool
	// If NoMedia is true, the proxy won't be used for media uploads/downloads
	NoMedia bool
}

// SetSOCKSProxy sets a SOCKS5 proxy to use for WhatsApp web websocket connections and media uploads/downloads.
//
// Same details as SetProxy apply, but using a different proxy for the websocket and media is not currently supported.
func (cli *Client) SetSOCKSProxy(px proxy.Dialer, opts ...SetProxyOptions) {
	var opt SetProxyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	transport := (http.DefaultTransport.(*http.Transport)).Clone()
	pxc := px.(proxy.ContextDialer)
	transport.DialContext = pxc.DialContext
	cli.setTransport(transport, opt)
}

func (cli *Client) setTransport(transport *http.Transport, opt SetProxyOptions) {
	if !opt.NoWebsocket {
		cli.preLoginHTTP.Transport = transport
		if !opt.OnlyLogin {
			cli.websocketHTTP.Transport = transport
		}
	}
	if !opt.NoMedia {
		cli.mediaHTTP.Transport = transport
	}
}

// SetMediaHTTPClient sets the HTTP client used to download media.
// This will overwrite any set proxy calls.
func (cli *Client) SetMediaHTTPClient(h *http.Client) {
	cli.mediaHTTP = h
}

// SetWebsocketHTTPClient sets the HTTP client used to establish the websocket connection for logged-in sessions.
// This will overwrite any set proxy calls.
func (cli *Client) SetWebsocketHTTPClient(h *http.Client) {
	cli.websocketHTTP = h
}

// SetPreLoginHTTPClient sets the HTTP client used to establish the websocket connection before login.
// This will overwrite any set proxy calls.
func (cli *Client) SetPreLoginHTTPClient(h *http.Client) {
	cli.preLoginHTTP = h
}

func (cli *Client) getSocketWaitChan() <-chan struct{} {
	cli.socketLock.RLock()
	ch := cli.socketWait
	cli.socketLock.RUnlock()
	return ch
}

func (cli *Client) closeSocketWaitChan() {
	cli.socketLock.Lock()
	close(cli.socketWait)
	cli.socketWait = make(chan struct{})
	cli.socketLock.Unlock()
}

func (cli *Client) getOwnID() types.JID {
	if cli == nil {
		return types.EmptyJID
	}
	return cli.Store.GetJID()
}

func (cli *Client) getOwnLID() types.JID {
	if cli == nil {
		return types.EmptyJID
	}
	return cli.Store.GetLID()
}

func (cli *Client) WaitForConnection(timeout time.Duration) bool {
	if cli == nil {
		return false
	}
	timeoutChan := time.After(timeout)
	cli.socketLock.RLock()
	for cli.socket == nil || !cli.socket.IsConnected() || !cli.IsLoggedIn() {
		ch := cli.socketWait
		cli.socketLock.RUnlock()
		select {
		case <-ch:
		case <-timeoutChan:
			return false
		case <-cli.expectedDisconnect.GetChan():
			return false
		}
		cli.socketLock.RLock()
	}
	cli.socketLock.RUnlock()
	return true
}

// Connect connects the client to the WhatsApp web websocket. After connection, it will either
// authenticate if there's data in the device store, or emit a QREvent to set up a new link.
func (cli *Client) Connect() error {
	return cli.ConnectContext(cli.BackgroundEventCtx)
}

func isRetryableConnectError(err error) bool {
	if exhttp.IsNetworkError(err) {
		return true
	}

	var statusErr socket.ErrWithStatusCode
	if errors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case 408, 500, 501, 502, 503, 504:
			return true
		}
	}

	return false
}

func (cli *Client) ConnectContext(ctx context.Context) error {
	if cli == nil {
		return ErrClientIsNil
	}

	cli.socketLock.Lock()
	defer cli.socketLock.Unlock()

	err := cli.unlockedConnect(ctx)
	if isRetryableConnectError(err) && cli.InitialAutoReconnect && cli.EnableAutoReconnect {
		cli.Log.Errorf("Initial connection failed but reconnecting in background (%v)", err)
		go cli.dispatchEvent(&events.Disconnected{})
		go cli.autoReconnect(ctx)
		return nil
	}
	return err
}

func (cli *Client) connect(ctx context.Context) error {
	cli.socketLock.Lock()
	defer cli.socketLock.Unlock()

	return cli.unlockedConnect(ctx)
}

func (cli *Client) unlockedConnect(ctx context.Context) error {
	if cli.socket != nil {
		if !cli.socket.IsConnected() {
			cli.unlockedDisconnect()
		} else {
			return ErrAlreadyConnected
		}
	}

	cli.resetExpectedDisconnect()
	client := cli.websocketHTTP
	if cli.Store.ID == nil {
		client = cli.preLoginHTTP
	}
	fs := socket.NewFrameSocket(cli.Log.Sub("Socket"), client)
	if cli.MessengerConfig != nil {
		fs.URL = cli.MessengerConfig.WebsocketURL
		fs.HTTPHeaders.Set("Origin", cli.MessengerConfig.BaseURL)
		fs.HTTPHeaders.Set("User-Agent", cli.MessengerConfig.UserAgent)
		fs.HTTPHeaders.Set("Cache-Control", "no-cache")
		fs.HTTPHeaders.Set("Pragma", "no-cache")
		// Messenger WebSocket 也需要 Sec-Fetch-* 头部以模拟真实浏览器
		fs.HTTPHeaders.Set("Sec-Fetch-Dest", "empty")
		fs.HTTPHeaders.Set("Sec-Fetch-Mode", "websocket")
		fs.HTTPHeaders.Set("Sec-Fetch-Site", "cross-site")
	} else {
		// 为 WhatsApp WebSocket 连接设置完整的浏览器头
		cli.setWebSocketHeaders(fs.HTTPHeaders)
	}
	if err := fs.Connect(ctx); err != nil {
		fs.Close(0)
		return err
	} else if err = cli.doHandshake(ctx, fs, *keys.NewKeyPair()); err != nil {
		fs.Close(0)
		return fmt.Errorf("noise handshake failed: %w", err)
	}
	go cli.keepAliveLoop(ctx, fs.Context())
	go cli.handlerQueueLoop(ctx, fs.Context())
	return nil
}

// IsLoggedIn returns true after the client is successfully connected and authenticated on WhatsApp.
func (cli *Client) IsLoggedIn() bool {
	return cli != nil && cli.isLoggedIn.Load()
}

func (cli *Client) onDisconnect(ctx context.Context, ns *socket.NoiseSocket, remote bool) {
	ns.Stop(false, false)
	cli.socketLock.Lock()
	defer cli.socketLock.Unlock()
	if cli.socket == ns {
		cli.socket = nil
		cli.clearResponseWaiters(xmlStreamEndNode)
		if !cli.isExpectedDisconnect() && (cli.forceAutoReconnect.Swap(false) || remote) {
			cli.Log.Debugf("Emitting Disconnected event")
			go cli.dispatchEvent(&events.Disconnected{})
			go cli.autoReconnect(ctx)
		} else if remote {
			cli.Log.Debugf("OnDisconnect() called, but it was expected, so not emitting event")
		} else {
			cli.Log.Debugf("OnDisconnect() called after manual disconnection")
		}
	} else {
		cli.Log.Debugf("Ignoring OnDisconnect on different socket")
	}
}

func (cli *Client) expectDisconnect() {
	cli.forceAutoReconnect.Store(false)
	cli.expectedDisconnect.Set()
}

func (cli *Client) resetExpectedDisconnect() {
	cli.forceAutoReconnect.Store(false)
	cli.expectedDisconnect.Clear()
}

func (cli *Client) isExpectedDisconnect() bool {
	return cli.expectedDisconnect.IsSet()
}

func (cli *Client) autoReconnect(ctx context.Context) {
	if !cli.EnableAutoReconnect || cli.Store.ID == nil {
		return
	}
	for {
		autoReconnectDelay := time.Duration(cli.AutoReconnectErrors) * 2 * time.Second
		cli.Log.Debugf("Automatically reconnecting after %v", autoReconnectDelay)
		cli.AutoReconnectErrors++
		if cli.expectedDisconnect.WaitTimeoutCtx(ctx, autoReconnectDelay) == nil {
			cli.Log.Debugf("Cancelling automatic reconnect due to expected disconnect")
			return
		} else if ctx.Err() != nil {
			cli.Log.Debugf("Cancelling automatic reconnect due to context cancellation")
			return
		}
		err := cli.connect(ctx)
		if errors.Is(err, ErrAlreadyConnected) {
			cli.Log.Debugf("Connect() said we're already connected after autoreconnect sleep")
			return
		} else if err != nil {
			if cli.expectedDisconnect.IsSet() {
				cli.Log.Debugf("Autoreconnect failed, but disconnect was expected, not reconnecting")
				return
			}
			cli.Log.Errorf("Error reconnecting after autoreconnect sleep: %v", err)
			if cli.AutoReconnectHook != nil && !cli.AutoReconnectHook(err) {
				cli.Log.Debugf("AutoReconnectHook returned false, not reconnecting")
				return
			}
		} else {
			return
		}
	}
}

// IsConnected checks if the client is connected to the WhatsApp web websocket.
// Note that this doesn't check if the client is authenticated. See the IsLoggedIn field for that.
func (cli *Client) IsConnected() bool {
	if cli == nil {
		return false
	}
	cli.socketLock.RLock()
	connected := cli.socket != nil && cli.socket.IsConnected()
	cli.socketLock.RUnlock()
	return connected
}

// Disconnect disconnects from the WhatsApp web websocket.
//
// This will not emit any events, the Disconnected event is only used when the
// connection is closed by the server or a network error.
func (cli *Client) Disconnect() {
	if cli == nil {
		return
	}
	cli.socketLock.Lock()
	cli.expectDisconnect()
	cli.unlockedDisconnect()
	cli.socketLock.Unlock()
	cli.clearDelayedMessageRequests()
	// 清除内存中的临时指纹（但保留数据库中的指纹，供下次使用）
	cli.pendingFingerprintLock.Lock()
	hadPending := cli.pendingFingerprint != nil
	cli.pendingFingerprint = nil
	cli.pendingFingerprintSaved.Store(false)
	cli.pendingFingerprintLock.Unlock()
	if hadPending && cli.Log != nil {
		cli.Log.Infof("[Fingerprint] Disconnect: cleared pending fingerprint from memory (database fingerprints preserved)")
	}
	// 清除会话地理信息缓存（重登时允许重新设置）
	cli.clearSessionGeoCache()
	// 注意：不清理 carrierInfo，因为它是外部传入的，可能需要在下次连接时继续使用
	// 如果业务层需要清理，可以显式调用 SetCarrierInfo("", "")
}

// ResetConnection disconnects from the WhatsApp web websocket and forces an automatic reconnection.
// This will not do anything if the socket is already disconnected or if EnableAutoReconnect is false.
func (cli *Client) ResetConnection() {
	if cli == nil {
		return
	}
	cli.socketLock.Lock()
	cli.forceAutoReconnect.Store(true)
	if cli.socket != nil {
		cli.socket.Stop(true, true)
		cli.clearResponseWaiters(xmlStreamEndNode)
	}
	cli.socketLock.Unlock()
}

// Disconnect closes the websocket connection.
func (cli *Client) unlockedDisconnect() {
	if cli.socket != nil {
		cli.socket.Stop(true, false)
		cli.socket = nil
		cli.clearResponseWaiters(xmlStreamEndNode)
	}
}

// Logout sends a request to unlink the device, then disconnects from the websocket and deletes the local device store.
//
// If the logout request fails, the disconnection and local data deletion will not happen either.
// If an error is returned, but you want to force disconnect/clear data, call Client.Disconnect() and Client.Store.Delete() manually.
//
// Note that this will not emit any events. The LoggedOut event is only used for external logouts
// (triggered by the user from the main device or by WhatsApp servers).
func (cli *Client) Logout(ctx context.Context) error {
	if cli == nil {
		return ErrClientIsNil
	} else if cli.MessengerConfig != nil {
		return errors.New("can't logout with Messenger credentials")
	}
	ownID := cli.getOwnID()
	if ownID.IsEmpty() {
		return ErrNotLoggedIn
	}
	_, err := cli.sendIQ(ctx, infoQuery{
		Namespace: "md",
		Type:      "set",
		To:        types.ServerJID,
		Content: []waBinary.Node{{
			Tag: "remove-companion-device",
			Attrs: waBinary.Attrs{
				"jid":    ownID,
				"reason": "user_initiated",
			},
		}},
	})
	if err != nil {
		return fmt.Errorf("error sending logout request: %w", err)
	}

	// 清除内存中的临时指纹
	cli.pendingFingerprintLock.Lock()
	hadPending := cli.pendingFingerprint != nil
	cli.pendingFingerprint = nil
	cli.pendingFingerprintSaved.Store(false)
	cli.pendingFingerprintLock.Unlock()
	if hadPending && cli.Log != nil {
		cli.Log.Infof("[Fingerprint] Logout: cleared pending fingerprint from memory")
	}
	// 注意：不清理 carrierInfo，因为它是外部传入的，可能需要在下次连接时继续使用
	// 如果业务层需要清理，可以显式调用 SetCarrierInfo("", "", "")

	// 删除实际 JID 的指纹（但保留电话号码 JID 的临时指纹，供下次配对使用）
	if container, ok := cli.Store.Container.(*sqlstore.Container); ok {
		if cli.Log != nil {
			cli.Log.Infof("[Fingerprint] Logout: deleting fingerprint for JID %s (temporary phone fingerprint preserved)", ownID.User)
		}
		if err := container.DeleteFingerprint(ctx, ownID); err != nil {
			cli.Log.Warnf("[Fingerprint] Failed to delete fingerprint for %s: %v", ownID.User, err)
			// 不阻止 logout 流程
		} else {
			cli.Log.Infof("[Fingerprint] Successfully deleted fingerprint for %s", ownID.User)
		}
	}

	cli.Disconnect()
	err = cli.Store.Delete(ctx)
	if err != nil {
		return fmt.Errorf("error deleting data from store: %w", err)
	}
	return nil
}

// AddEventHandler registers a new function to receive all events emitted by this client.
//
// The returned integer is the event handler ID, which can be passed to RemoveEventHandler to remove it.
//
// All registered event handlers will receive all events. You should use a type switch statement to
// filter the events you want:
//
//	func myEventHandler(evt interface{}) {
//		switch v := evt.(type) {
//		case *events.Message:
//			fmt.Println("Received a message!")
//		case *events.Receipt:
//			fmt.Println("Received a receipt!")
//		}
//	}
//
// If you want to access the Client instance inside the event handler, the recommended way is to
// wrap the whole handler in another struct:
//
//	type MyClient struct {
//		WAClient *whatsmeow.Client
//		eventHandlerID uint32
//	}
//
//	func (mycli *MyClient) register() {
//		mycli.eventHandlerID = mycli.WAClient.AddEventHandler(mycli.myEventHandler)
//	}
//
//	func (mycli *MyClient) myEventHandler(evt interface{}) {
//		// Handle event and access mycli.WAClient
//	}
func (cli *Client) AddEventHandler(handler EventHandler) uint32 {
	return cli.AddEventHandlerWithSuccessStatus(func(evt any) bool {
		handler(evt)
		return true
	})
}

func (cli *Client) AddEventHandlerWithSuccessStatus(handler EventHandlerWithSuccessStatus) uint32 {
	nextID := atomic.AddUint32(&nextHandlerID, 1)
	cli.eventHandlersLock.Lock()
	cli.eventHandlers = append(cli.eventHandlers, wrappedEventHandler{handler, nextID})
	cli.eventHandlersLock.Unlock()
	return nextID
}

// RemoveEventHandler removes a previously registered event handler function.
// If the function with the given ID is found, this returns true.
//
// N.B. Do not run this directly from an event handler. That would cause a deadlock because the
// event dispatcher holds a read lock on the event handler list, and this method wants a write lock
// on the same list. Instead run it in a goroutine:
//
//	func (mycli *MyClient) myEventHandler(evt interface{}) {
//		if noLongerWantEvents {
//			go mycli.WAClient.RemoveEventHandler(mycli.eventHandlerID)
//		}
//	}
func (cli *Client) RemoveEventHandler(id uint32) bool {
	cli.eventHandlersLock.Lock()
	defer cli.eventHandlersLock.Unlock()
	for index := range cli.eventHandlers {
		if cli.eventHandlers[index].id == id {
			if index == 0 {
				cli.eventHandlers[0].fn = nil
				cli.eventHandlers = cli.eventHandlers[1:]
				return true
			} else if index < len(cli.eventHandlers)-1 {
				copy(cli.eventHandlers[index:], cli.eventHandlers[index+1:])
			}
			cli.eventHandlers[len(cli.eventHandlers)-1].fn = nil
			cli.eventHandlers = cli.eventHandlers[:len(cli.eventHandlers)-1]
			return true
		}
	}
	return false
}

// RemoveEventHandlers removes all event handlers that have been registered with AddEventHandler
func (cli *Client) RemoveEventHandlers() {
	cli.eventHandlersLock.Lock()
	cli.eventHandlers = make([]wrappedEventHandler, 0, 1)
	cli.eventHandlersLock.Unlock()
}

func (cli *Client) handleFrame(ctx context.Context, data []byte) {
	decompressed, err := waBinary.Unpack(data)
	if err != nil {
		cli.Log.Warnf("Failed to decompress frame: %v", err)
		cli.Log.Debugf("Errored frame hex: %s", hex.EncodeToString(data))
		return
	}
	node, err := waBinary.Unmarshal(decompressed)
	if err != nil {
		cli.Log.Warnf("Failed to decode node in frame: %v", err)
		cli.Log.Debugf("Errored frame hex: %s", hex.EncodeToString(decompressed))
		return
	}
	cli.recvLog.Debugf("%s", node.XMLString())
	if node.Tag == "xmlstreamend" {
		if !cli.isExpectedDisconnect() {
			cli.Log.Warnf("Received stream end frame")
		}
		// TODO should we do something else?
	} else if cli.receiveResponse(ctx, node) {
		// handled
	} else if _, ok := cli.nodeHandlers[node.Tag]; ok {
		select {
		case cli.handlerQueue <- node:
		case <-ctx.Done():
		default:
			cli.Log.Warnf("Handler queue is full, message ordering is no longer guaranteed")
			go func() {
				select {
				case cli.handlerQueue <- node:
				case <-ctx.Done():
				}
			}()
		}
	} else if node.Tag != "ack" {
		cli.Log.Debugf("Didn't handle WhatsApp node %s", node.Tag)
	}
}

func (cli *Client) handlerQueueLoop(evtCtx, connCtx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	ticker.Stop()
	cli.Log.Debugf("Starting handler queue loop")
Loop:
	for {
		select {
		case node := <-cli.handlerQueue:
			doneChan := make(chan struct{}, 1)
			start := time.Now()
			go func() {
				cli.nodeHandlers[node.Tag](evtCtx, node)
				duration := time.Since(start)
				doneChan <- struct{}{}
				if duration > 5*time.Second {
					cli.Log.Warnf("Node handling took %s for %s", duration, node.XMLString())
				}
			}()
			ticker.Reset(30 * time.Second)
			for i := 0; i < 10; i++ {
				select {
				case <-doneChan:
					ticker.Stop()
					continue Loop
				case <-ticker.C:
					cli.Log.Warnf("Node handling is taking long for %s (started %s ago)", node.XMLString(), time.Since(start))
				}
			}
			cli.Log.Warnf("Continuing handling of %s in background as it's taking too long", node.XMLString())
			ticker.Stop()
		case <-connCtx.Done():
			cli.Log.Debugf("Closing handler queue loop")
			return
		}
	}
}

func (cli *Client) sendNodeAndGetData(ctx context.Context, node waBinary.Node) ([]byte, error) {
	if cli == nil {
		return nil, ErrClientIsNil
	}
	cli.socketLock.RLock()
	sock := cli.socket
	cli.socketLock.RUnlock()
	if sock == nil {
		return nil, ErrNotConnected
	}

	payload, err := waBinary.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node: %w", err)
	}

	cli.sendLog.Debugf("%s", node.XMLString())
	return payload, sock.SendFrame(ctx, payload)
}

func (cli *Client) sendNode(ctx context.Context, node waBinary.Node) error {
	_, err := cli.sendNodeAndGetData(ctx, node)
	return err
}

func (cli *Client) dispatchEvent(evt any) (handlerFailed bool) {
	cli.eventHandlersLock.RLock()
	defer func() {
		cli.eventHandlersLock.RUnlock()
		err := recover()
		if err != nil {
			cli.Log.Errorf("Event handler panicked while handling a %T: %v\n%s", evt, err, debug.Stack())
		}
	}()
	for _, handler := range cli.eventHandlers {
		if !handler.fn(evt) {
			return true
		}
	}
	return false
}

// ParseWebMessage parses a WebMessageInfo object into *events.Message to match what real-time messages have.
//
// The chat JID can be found in the Conversation data:
//
//	chatJID, err := types.ParseJID(conv.GetId())
//	for _, historyMsg := range conv.GetMessages() {
//		evt, err := cli.ParseWebMessage(chatJID, historyMsg.GetMessage())
//		yourNormalEventHandler(evt)
//	}
func (cli *Client) ParseWebMessage(chatJID types.JID, webMsg *waWeb.WebMessageInfo) (*events.Message, error) {
	var err error
	if chatJID.IsEmpty() {
		chatJID, err = types.ParseJID(webMsg.GetKey().GetRemoteJID())
		if err != nil {
			return nil, fmt.Errorf("no chat JID provided and failed to parse remote JID: %w", err)
		}
	}
	info := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     chatJID,
			IsFromMe: webMsg.GetKey().GetFromMe(),
			IsGroup:  chatJID.Server == types.GroupServer,
		},
		ID:        webMsg.GetKey().GetID(),
		PushName:  webMsg.GetPushName(),
		Timestamp: time.Unix(int64(webMsg.GetMessageTimestamp()), 0),
	}
	if info.IsFromMe {
		info.Sender = cli.getOwnID().ToNonAD()
		if info.Sender.IsEmpty() {
			return nil, ErrNotLoggedIn
		}
	} else if chatJID.Server == types.DefaultUserServer || chatJID.Server == types.HiddenUserServer || chatJID.Server == types.NewsletterServer {
		info.Sender = chatJID
	} else if webMsg.GetParticipant() != "" {
		info.Sender, err = types.ParseJID(webMsg.GetParticipant())
	} else if webMsg.GetKey().GetParticipant() != "" {
		info.Sender, err = types.ParseJID(webMsg.GetKey().GetParticipant())
	} else {
		return nil, fmt.Errorf("couldn't find sender of message %s", info.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse sender of message %s: %v", info.ID, err)
	}
	if pk := webMsg.GetCommentMetadata().GetCommentParentKey(); pk != nil {
		info.MsgMetaInfo.ThreadMessageID = pk.GetID()
		info.MsgMetaInfo.ThreadMessageSenderJID, _ = types.ParseJID(pk.GetParticipant())
	}
	evt := &events.Message{
		RawMessage:   webMsg.GetMessage(),
		SourceWebMsg: webMsg,
		Info:         info,
	}
	evt.UnwrapRaw()
	if evt.Message.GetProtocolMessage().GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT {
		evt.Info.ID = evt.Message.GetProtocolMessage().GetKey().GetID()
		evt.Message = evt.Message.GetProtocolMessage().GetEditedMessage()
	}
	return evt, nil
}

func (cli *Client) StoreLIDPNMapping(ctx context.Context, first, second types.JID) {
	var lid, pn types.JID
	if first.Server == types.HiddenUserServer && second.Server == types.DefaultUserServer {
		lid = first
		pn = second
	} else if first.Server == types.DefaultUserServer && second.Server == types.HiddenUserServer {
		lid = second
		pn = first
	} else {
		return
	}
	err := cli.Store.LIDs.PutLIDMapping(ctx, lid, pn)
	if err != nil {
		cli.Log.Errorf("Failed to store LID-PN mapping for %s -> %s: %v", lid, pn, err)
	}
}

const (
	DayMs    = 24 * 60 * 60 * 1000
	WeekMs   = 7 * DayMs
	OffsetMs = 3 * DayMs
)

func getUnifiedSessionID() string {
	now := time.Now().UnixMilli()
	id := (now + int64(OffsetMs)) % int64(WeekMs)
	return strconv.FormatInt(id, 10)
}

func (cli *Client) sendUnifiedSession(ctx context.Context) {
	if cli == nil || !cli.IsConnected() {
		return
	}

	sessionID := getUnifiedSessionID()

	node := waBinary.Node{
		Tag: "ib",
		Content: []waBinary.Node{
			{
				Tag: "unified_session",
				Attrs: waBinary.Attrs{
					"id": sessionID,
				},
			},
		},
	}

	err := cli.sendNode(ctx, node)
	if err != nil {
		cli.Log.Debugf("Failed to send unified_session telemetry: %v", err)
	}
}
