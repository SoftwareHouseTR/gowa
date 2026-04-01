package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Type definitions
type ExtractedMedia struct {
	MediaPath string `json:"media_path"`
	MimeType  string `json:"mime_type"`
	Caption   string `json:"caption"`
}

// Global variables
var (
	globalStateMu sync.RWMutex
	cli           *whatsmeow.Client
	db            *sqlstore.Container // Add global database reference for cleanup
	keysDB        *sqlstore.Container
	deviceManager *DeviceManager
	log           waLog.Logger
	startupTime   = time.Now().Unix()
)

// syncKeysDevice ensures the given device exists in the keys database.
// It only adds/updates the specific device rather than deleting others,
// so that multiple devices can coexist in the keys store.
func syncKeysDevice(ctx context.Context, device *store.Device, keysDB *sqlstore.Container) {
	if keysDB == nil || device == nil || device.ID == nil {
		return
	}

	devs, err := keysDB.GetAllDevices(ctx)
	if err != nil {
		log.Errorf("Failed to get devices from keys store: %v", err)
		return
	}

	for _, d := range devs {
		if d.ID != nil && *d.ID == *device.ID {
			return // device already exists in keys store
		}
	}

	// Device not found in keys store, add it
	if err := keysDB.PutDevice(ctx, device); err != nil {
		log.Errorf("Failed to sync device %v to keys store: %v", device.ID, err)
	}
}

// InitWaCLI initializes the WhatsApp client
func InitWaCLI(ctx context.Context, storeContainer, keysStoreContainer *sqlstore.Container, chatStorageRepo domainChatStorage.IChatStorageRepository) *whatsmeow.Client {
	device, err := storeContainer.GetFirstDevice(ctx)
	if err != nil {
		log.Errorf("Failed to get device: %v", err)
		panic(err)
	}

	if device == nil {
		log.Errorf("No device found")
		panic("No device found")
	}

	// Configure device properties
	osName := fmt.Sprintf("%s %s", config.AppOs, config.AppVersion)
	store.DeviceProps.PlatformType = &config.AppPlatform
	store.DeviceProps.Os = &osName

	// Keep references for global state update after client creation
	primaryDB := storeContainer
	keysContainer := keysStoreContainer

	// Configure a separated database for accelerating encryption caching
	if keysContainer != nil && device.ID != nil {
		innerStore := sqlstore.NewSQLStore(keysStoreContainer, *device.ID)

		syncKeysDevice(ctx, device, keysContainer)
		device.Identities = innerStore
		device.Sessions = innerStore
		device.PreKeys = innerStore
		device.SenderKeys = innerStore
		device.MsgSecrets = innerStore
		device.PrivacyTokens = innerStore
	}

	instanceID := ""
	if device.ID != nil {
		instanceID = device.ID.String()
	}

	// Create and configure the client with filtered logging to avoid noisy reconnection EOF errors
	baseLogger := waLog.Stdout("Client", config.WhatsappLogLevel, true)
	client := whatsmeow.NewClient(device, newFilteredLogger(baseLogger))
	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true
	client.UseRetryMessageStore = true

	deviceRepo := newDeviceChatStorage(instanceID, chatStorageRepo)
	instance := NewDeviceInstance(instanceID, client, deviceRepo)

	client.AddEventHandler(func(rawEvt interface{}) {
		handler(ctx, instance, rawEvt)
	})

	// Register device instance in the manager for multi-device awareness
	// Use EnsureDefault to avoid creating duplicates when a device with matching JID already exists
	if device.ID != nil {
		instanceID = device.ID.String()
	}
	dm := InitializeDeviceManager(storeContainer, keysStoreContainer, deviceRepo)
	if dm != nil && instanceID != "" {
		dm.EnsureDefault(instance)
		instance.SetOnLoggedOut(func(deviceID string) {
			dm.RemoveDevice(deviceID)
		})
	}

	globalStateMu.Lock()
	cli = client
	db = primaryDB
	keysDB = keysContainer
	globalStateMu.Unlock()

	return client
}
