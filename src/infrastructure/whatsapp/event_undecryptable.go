package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// sessionResetTracker keeps track of recent session resets to avoid
// resetting the same sender's session in a tight loop.
// The key is "deviceJID:senderJID", the value is the last reset time.
var (
	sessionResetMu      sync.Mutex
	sessionResetHistory = make(map[string]time.Time)
)

// sessionResetCooldown is the minimum interval between two session resets
// for the same sender on the same device.
const sessionResetCooldown = 5 * time.Minute

// handleUndecryptableMessage handles messages that could not be decrypted.
// When the failure is a genuine decryption error (not a missing ciphertext),
// it deletes the local Signal session for the sender so that the next message
// exchange triggers a fresh PreKey handshake.
func handleUndecryptableMessage(ctx context.Context, evt *events.UndecryptableMessage, deviceID string, client *whatsmeow.Client) {
	sender := evt.Info.Sender.ToNonAD().String()

	// If the ciphertext was never sent to us (IsUnavailable), there is no
	// corrupted session to reset — the sender simply didn't encrypt for
	// this device. whatsmeow already requests a retry in this case.
	if evt.IsUnavailable {
		logrus.WithFields(logrus.Fields{
			"device_id":  deviceID,
			"sender":     sender,
			"message_id": evt.Info.ID,
		}).Debug("[DECRYPT] Unavailable message (no ciphertext for this device)")
		return
	}

	// DecryptFailHide means the sender indicated this failure should be
	// silent (e.g. intentionally unavailable media). Don't reset sessions.
	if evt.DecryptFailMode == events.DecryptFailHide {
		return
	}

	if client == nil || client.Store == nil || client.Store.Sessions == nil {
		return
	}

	// Throttle: only reset once per sender per cooldown window.
	resetKey := fmt.Sprintf("%s:%s", deviceID, sender)
	sessionResetMu.Lock()
	if last, ok := sessionResetHistory[resetKey]; ok && time.Since(last) < sessionResetCooldown {
		sessionResetMu.Unlock()
		return
	}
	sessionResetHistory[resetKey] = time.Now()
	sessionResetMu.Unlock()

	// Delete all Signal sessions for this sender's phone number.
	// This forces a fresh PreKey negotiation on the next message exchange.
	phone := evt.Info.Sender.User
	if err := client.Store.Sessions.DeleteAllSessions(ctx, phone); err != nil {
		logrus.WithFields(logrus.Fields{
			"device_id": deviceID,
			"sender":    sender,
			"error":     err,
		}).Error("[DECRYPT] Failed to reset Signal sessions")
		return
	}

	logrus.WithFields(logrus.Fields{
		"device_id": deviceID,
		"sender":    sender,
		"chat":      evt.Info.Chat.ToNonAD().String(),
	}).Warn("[DECRYPT] Signal sessions reset for sender — next exchange will use fresh PreKey handshake")
}
