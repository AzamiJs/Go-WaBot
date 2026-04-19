package handler

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"mini_wabot/lib/db"
	"mini_wabot/lib/simple"
	"mini_wabot/lib/utils"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

var waLinkRegex = regexp.MustCompile(`(?i)chat\.whatsapp\.com/(?:invite/)?([0-9a-zA-Z]{20,26})`)

func AntiLink(client *whatsmeow.Client, m *simple.Message) {
	if m.Text == "" || !m.IsGroup {
		return
	}

	database := db.Get()
	chat := database.GetOrCreateChat(m.Chat)
	if !chat.Antilink {
		return
	}

	if !waLinkRegex.MatchString(m.Text) {
		return
	}

	groupCode, _ := client.GetGroupInviteLink(context.Background(), m.ChatJID, false)

	if groupCode != "" && strings.Contains(m.Text, groupCode) {
		client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			Conversation: proto.String("Enviaste un *enlace* perteneciente a este *grupo*"),
		})
		return
	}

	targetJid := utils.ResolveLidToRealJid(m.Sender, cachedGroupMeta(client, m.Chat))

	client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
		ProtocolMessage: &waProto.ProtocolMessage{
			Key:  m.Key,
			Type: waProto.ProtocolMessage_REVOKE.Enum(),
		},
	})

	caption := fmt.Sprintf("Anti Enlaces\n\n@%s enlace prohibido detectado", strings.Split(targetJid, "@")[0])
	client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(caption),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: []string{targetJid},
			},
		},
	})

	targetJID, err := types.ParseJID(targetJid)
	if err != nil {
		return
	}
	client.UpdateGroupParticipants(context.Background(), m.ChatJID, []types.JID{targetJID}, whatsmeow.ParticipantChangeRemove)
}
