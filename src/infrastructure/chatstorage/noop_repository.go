package chatstorage

import (
	"context"
	"time"

	domainChatStorage "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/chatstorage"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// NoopRepository is a no-op implementation of IChatStorageRepository
// used when chat storage is disabled via WHATSAPP_CHAT_STORAGE=false.
type NoopRepository struct{}

func NewNoopRepository() domainChatStorage.IChatStorageRepository {
	return &NoopRepository{}
}

func (r *NoopRepository) CreateMessage(_ context.Context, _ *events.Message) error { return nil }
func (r *NoopRepository) CreateIncomingCallRecord(_ context.Context, _ *events.CallOffer, _ bool) error {
	return nil
}
func (r *NoopRepository) StoreChat(_ *domainChatStorage.Chat) error        { return nil }
func (r *NoopRepository) GetChat(_ string) (*domainChatStorage.Chat, error) { return nil, nil }
func (r *NoopRepository) GetChatByDevice(_, _ string) (*domainChatStorage.Chat, error) {
	return nil, nil
}
func (r *NoopRepository) GetChats(_ *domainChatStorage.ChatFilter) ([]*domainChatStorage.Chat, error) {
	return nil, nil
}
func (r *NoopRepository) DeleteChat(_ string) error            { return nil }
func (r *NoopRepository) DeleteChatByDevice(_, _ string) error { return nil }

func (r *NoopRepository) StoreMessage(_ *domainChatStorage.Message) error      { return nil }
func (r *NoopRepository) StoreMessagesBatch(_ []*domainChatStorage.Message) error { return nil }
func (r *NoopRepository) GetMessageByID(_ string) (*domainChatStorage.Message, error) {
	return nil, nil
}
func (r *NoopRepository) GetMessages(_ *domainChatStorage.MessageFilter) ([]*domainChatStorage.Message, error) {
	return nil, nil
}
func (r *NoopRepository) SearchMessages(_, _, _ string, _ int) ([]*domainChatStorage.Message, error) {
	return nil, nil
}
func (r *NoopRepository) DeleteMessage(_, _ string) error            { return nil }
func (r *NoopRepository) DeleteMessageByDevice(_, _, _ string) error { return nil }
func (r *NoopRepository) StoreSentMessageWithContext(_ context.Context, _ string, _ string, _ string, _ string, _ time.Time, _ *waE2E.Message) error {
	return nil
}

func (r *NoopRepository) GetChatMessageCount(_ string) (int64, error)        { return 0, nil }
func (r *NoopRepository) GetChatMessageCountByDevice(_, _ string) (int64, error) { return 0, nil }
func (r *NoopRepository) GetTotalMessageCount() (int64, error)               { return 0, nil }
func (r *NoopRepository) GetTotalChatCount() (int64, error)                  { return 0, nil }
func (r *NoopRepository) GetFilteredChatCount(_ *domainChatStorage.ChatFilter) (int64, error) {
	return 0, nil
}
func (r *NoopRepository) GetChatNameWithPushName(_ types.JID, _, _, _ string) string { return "" }
func (r *NoopRepository) GetChatNameWithPushNameByDevice(_ string, _ types.JID, _, _, _ string) string {
	return ""
}
func (r *NoopRepository) GetStorageStatistics() (int64, int64, error) { return 0, 0, nil }

func (r *NoopRepository) TruncateAllChats() error                    { return nil }
func (r *NoopRepository) TruncateAllDataWithLogging(_ string) error  { return nil }
func (r *NoopRepository) DeleteDeviceData(_ string) error            { return nil }

func (r *NoopRepository) SaveDeviceRecord(_ *domainChatStorage.DeviceRecord) error { return nil }
func (r *NoopRepository) ListDeviceRecords() ([]*domainChatStorage.DeviceRecord, error) {
	return nil, nil
}
func (r *NoopRepository) GetDeviceRecord(_ string) (*domainChatStorage.DeviceRecord, error) {
	return nil, nil
}
func (r *NoopRepository) DeleteDeviceRecord(_ string) error { return nil }

func (r *NoopRepository) InitializeSchema() error { return nil }
