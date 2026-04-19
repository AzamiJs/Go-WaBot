package owner

import (
	"context"
	"fmt"
	"strings"

	"mini_wabot/lib/db"
	"mini_wabot/lib/simple"
	"mini_wabot/plugin"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

func Plugins() []*plugin.Plugin {
	return []*plugin.Plugin{listUsersPlugin}
}

var listUsersPlugin = &plugin.Plugin{
	Command:     []string{"listusers", "usuarios", "database"},
	Description: "Muestra la lista de usuarios registrados en la base de datos",
	Category:    "owner",
	IsOwner:     true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		database := db.Get()
		database.RLock()
		users := make(map[string]*db.UserData, len(database.Users))
		for k, v := range database.Users {
			users[k] = v
		}
		database.RUnlock()

		total := len(users)
		if total == 0 {
			return m.Reply("No hay usuarios registrados en la base de datos aún.")
		}

		message := fmt.Sprintf("*Total de usuarios:* %d\n\n", total)
		var jids []string
		for jid := range users {
			jids = append(jids, jid)
		}

		limit := 50
		if total < limit {
			limit = total
		}
		topUsers := jids[:limit]
		var mentions []string

		for i, jid := range topUsers {
			user := users[jid]
			name := user.Name
			if name == "" {
				name = "Usuario"
			}
			if len(name) > 15 {
				name = name[:15]
			}
			tag := strings.Split(jid, "@")[0]
			message += fmt.Sprintf(" %d. ◦ @%s (%s)\n", i+1, tag, name)
			mentions = append(mentions, jid+"@s.whatsapp.net")
		}

		if total > 50 {
			message += fmt.Sprintf("\n_... y %d usuarios más._", total-50)
		}

		_, err := client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(message),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      proto.String(m.ID),
					Participant:   proto.String(m.Sender),
					QuotedMessage: m.Msg,
					MentionedJID:  mentions,
				},
			},
		})
		return err
	},
}
