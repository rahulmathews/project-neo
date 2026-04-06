package whatsapp

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"project-neo/workers/internal/store"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// Handler processes incoming WhatsApp message events.
type Handler struct {
	jidMap    map[string]uuid.UUID
	srcMap    map[string]uuid.UUID
	msgWriter *store.MessageWriter
	srcReader *store.GroupSourceReader
	logger    *slog.Logger
	wg        *sync.WaitGroup
}

func NewHandler(
	jidMap map[string]uuid.UUID,
	srcMap map[string]uuid.UUID,
	msgWriter *store.MessageWriter,
	srcReader *store.GroupSourceReader,
	logger *slog.Logger,
	wg *sync.WaitGroup,
) *Handler {
	return &Handler{
		jidMap:    jidMap,
		srcMap:    srcMap,
		msgWriter: msgWriter,
		srcReader: srcReader,
		logger:    logger,
		wg:        wg,
	}
}

// Handle processes a single WhatsApp message event.
func (h *Handler) Handle(evt *events.Message) {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.process(evt)
	}()
}

func (h *Handler) process(evt *events.Message) {
	chatJID := evt.Info.Chat.String()

	groupID, ok := h.jidMap[chatJID]
	if !ok {
		return // not a monitored group — skip silently
	}

	text := extractText(evt.Message)
	if text == "" {
		return // image/sticker/reaction with no text — skip
	}

	// Anti-detection: random jitter 300ms–2s before writing to DB.
	jitter := time.Duration(300+rand.IntN(1700)) * time.Millisecond
	time.Sleep(jitter)

	ctx := context.Background()

	msgID := evt.Info.ID
	var sourceMessageID *string
	if msgID != "" {
		sourceMessageID = &msgID
	}

	sender := evt.Info.Sender.String()
	stored, err := h.msgWriter.Write(
		ctx,
		groupID,
		h.srcMap[chatJID],
		sourceMessageID,
		&sender,
		text,
		evt.Info.Timestamp,
	)
	if err != nil {
		h.logger.Error("failed to store message",
			"group_id", groupID,
			"source_message_id", msgID,
			"error", err,
		)
		return
	}

	if stored {
		h.logger.Info("message stored",
			"group_id", groupID,
			"source_message_id", msgID,
		)
		srcID := h.srcMap[chatJID]
		h.srcReader.TouchLastParsedAt(ctx, srcID)
	}
}

// extractText returns the plain text body from a WhatsApp message proto.
func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if text := msg.GetConversation(); text != "" {
		return text
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}
