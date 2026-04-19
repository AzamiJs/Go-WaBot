package simple

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type QuotedMessage struct {
	ID         string
	Chat       string
	Sender     string
	FromMe     bool
	Type       string
	Mimetype   string
	IsMedia    bool
	IsAnimated bool
	Text       string
	RawMsg     *waProto.Message
	client     *whatsmeow.Client
}

func (q *QuotedMessage) Download() ([]byte, error) {
	if q.RawMsg == nil {
		return nil, nil
	}
	return downloadMedia(q.client, q.RawMsg, q.Type)
}

func (q *QuotedMessage) Delete(chatJid types.JID) error {
	key := &waProto.MessageKey{
		RemoteJID: proto.String(q.Chat),
		FromMe:    proto.Bool(q.FromMe),
		ID:        proto.String(q.ID),
	}
	_, err := q.client.SendMessage(context.Background(), chatJid, &waProto.Message{
		ProtocolMessage: &waProto.ProtocolMessage{
			Key:  key,
			Type: waProto.ProtocolMessage_REVOKE.Enum(),
		},
	})
	return err
}

type Message struct {
	ID           string
	Chat         string
	ChatJID      types.JID
	IsGroup      bool
	FromMe       bool
	Sender       string
	SenderJID    types.JID
	PushName     string
	Type         string
	Text         string
	Prefix       string
	Command      string
	Args         []string
	IsMedia      bool
	Mimetype     string
	IsAnimated   bool
	IsBot        bool
	Key          *waProto.MessageKey
	Msg          *waProto.Message
	Quoted       *QuotedMessage
	MentionedJID []string
	client       *whatsmeow.Client
	raw          *events.Message
}

func (m *Message) Reply(text string) error {
	_, err := m.client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(m.ID),
				Participant:   proto.String(m.Sender),
				QuotedMessage: m.Msg,
				MentionedJID:  []string{m.Sender},
			},
		},
	})
	return err
}

func (m *Message) Download() ([]byte, error) {
	return downloadMedia(m.client, m.Msg, m.Type)
}

func downloadMedia(client *whatsmeow.Client, msg *waProto.Message, msgType string) ([]byte, error) {
	ctx := context.Background()
	switch msgType {
	case "imageMessage":
		return client.Download(ctx, msg.GetImageMessage())
	case "videoMessage":
		return client.Download(ctx, msg.GetVideoMessage())
	case "audioMessage":
		return client.Download(ctx, msg.GetAudioMessage())
	case "stickerMessage":
		return client.Download(ctx, msg.GetStickerMessage())
	case "documentMessage":
		return client.Download(ctx, msg.GetDocumentMessage())
	}
	return nil, nil
}

var prefixRegex = []string{"°", "•", "π", "÷", "×", "¶", "∆", "£", "¢", "€", "¥", "®", "™", "+", "✓", "_", "=", "|", "~", "!", "?", "@", "#", "$", "%", "^", "&", ".", "©", "^"}

func hasPrefixChar(s string) (string, bool) {
	for _, p := range prefixRegex {
		if strings.HasPrefix(s, p) {
			return p, true
		}
	}
	return "", false
}

func getContentType(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Conversation != nil {
		return "conversation"
	}
	if msg.ExtendedTextMessage != nil {
		return "extendedTextMessage"
	}
	if msg.ImageMessage != nil {
		return "imageMessage"
	}
	if msg.VideoMessage != nil {
		return "videoMessage"
	}
	if msg.AudioMessage != nil {
		return "audioMessage"
	}
	if msg.StickerMessage != nil {
		return "stickerMessage"
	}
	if msg.DocumentMessage != nil {
		return "documentMessage"
	}
	if msg.ButtonsResponseMessage != nil {
		return "buttonsResponseMessage"
	}
	if msg.ListResponseMessage != nil {
		return "listResponseMessage"
	}
	if msg.TemplateButtonReplyMessage != nil {
		return "templateButtonReplyMessage"
	}
	return ""
}

func extractText(msg *waProto.Message, msgType string) string {
	switch msgType {
	case "conversation":
		return msg.GetConversation()
	case "extendedTextMessage":
		return msg.GetExtendedTextMessage().GetText()
	case "imageMessage":
		return msg.GetImageMessage().GetCaption()
	case "videoMessage":
		return msg.GetVideoMessage().GetCaption()
	case "buttonsResponseMessage":
		return msg.GetButtonsResponseMessage().GetSelectedButtonID() // FIX: Id -> ID
	case "listResponseMessage":
		return msg.GetListResponseMessage().GetSingleSelectReply().GetSelectedRowID()
	case "templateButtonReplyMessage":
		return msg.GetTemplateButtonReplyMessage().GetSelectedID()
	}
	return ""
}

func BuildMessage(client *whatsmeow.Client, evt *events.Message) *Message {
	m := &Message{
		client: client,
		raw:    evt,
		Key: &waProto.MessageKey{
			RemoteJID: proto.String(evt.Info.Chat.String()),
			FromMe:    proto.Bool(evt.Info.IsFromMe),
			ID:        proto.String(evt.Info.ID),
		},
	}

	m.ID = evt.Info.ID
	m.Chat = evt.Info.Chat.String()
	m.ChatJID = evt.Info.Chat
	m.IsGroup = evt.Info.IsGroup
	m.FromMe = evt.Info.IsFromMe
	m.PushName = evt.Info.PushName
	m.Msg = evt.Message

	if evt.Info.IsFromMe {
		m.Sender = client.Store.ID.String()
		m.SenderJID = *client.Store.ID
	} else if evt.Info.IsGroup {
		m.Sender = evt.Info.Sender.String()
		m.SenderJID = evt.Info.Sender
	} else {
		m.Sender = evt.Info.Chat.String()
		m.SenderJID = evt.Info.Chat
	}

	m.Type = getContentType(evt.Message)
	m.Text = extractText(evt.Message, m.Type)

	if m.Text != "" {
		if p, ok := hasPrefixChar(m.Text); ok {
			m.Prefix = p
		}
		body := strings.TrimSpace(strings.TrimPrefix(m.Text, m.Prefix))
		parts := strings.Fields(body)
		if len(parts) > 0 {
			m.Command = strings.ToLower(parts[0])
			m.Args = parts[1:]
		}
	}

	switch m.Type {
	case "imageMessage":
		m.IsMedia = true
		m.Mimetype = evt.Message.GetImageMessage().GetMimetype()
	case "videoMessage":
		m.IsMedia = true
		m.Mimetype = evt.Message.GetVideoMessage().GetMimetype()
	case "audioMessage":
		m.IsMedia = true
		m.Mimetype = evt.Message.GetAudioMessage().GetMimetype()
	case "stickerMessage":
		m.IsMedia = true
		m.Mimetype = "image/webp"
		m.IsAnimated = evt.Message.GetStickerMessage().GetIsAnimated()
	}

	ctx := getContextInfo(evt.Message, m.Type)
	if ctx != nil {
		m.MentionedJID = ctx.GetMentionedJID()
		if ctx.QuotedMessage != nil {
			q := &QuotedMessage{
				ID:     ctx.GetStanzaID(),
				Chat:   m.Chat,
				Sender: ctx.GetParticipant(),
				RawMsg: ctx.QuotedMessage,
				client: client,
			}
			q.Type = getContentType(ctx.QuotedMessage)
			q.Text = extractText(ctx.QuotedMessage, q.Type)
			switch q.Type {
			case "imageMessage":
				q.IsMedia = true
				q.Mimetype = ctx.QuotedMessage.GetImageMessage().GetMimetype()
			case "videoMessage":
				q.IsMedia = true
				q.Mimetype = ctx.QuotedMessage.GetVideoMessage().GetMimetype()
			case "audioMessage":
				q.IsMedia = true
				q.Mimetype = ctx.QuotedMessage.GetAudioMessage().GetMimetype()
			case "stickerMessage":
				q.IsMedia = true
				q.Mimetype = "image/webp"
			}
			m.Quoted = q
		}
	}

	return m
}

func getContextInfo(msg *waProto.Message, msgType string) *waProto.ContextInfo {
	switch msgType {
	case "extendedTextMessage":
		return msg.GetExtendedTextMessage().GetContextInfo()
	case "imageMessage":
		return msg.GetImageMessage().GetContextInfo()
	case "videoMessage":
		return msg.GetVideoMessage().GetContextInfo()
	case "audioMessage":
		return msg.GetAudioMessage().GetContextInfo()
	case "documentMessage":
		return msg.GetDocumentMessage().GetContextInfo()
	}
	return nil
}
