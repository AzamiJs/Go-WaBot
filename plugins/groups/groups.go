package groups

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mini_wabot/lib/db"
	"mini_wabot/lib/simple"
	"mini_wabot/plugin"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func Plugins() []*plugin.Plugin {
	return []*plugin.Plugin{hidetagPlugin, kickPlugin, promotePlugin, demotePlugin, onOffPlugin, changesPlugin}
}

var hidetagPlugin = &plugin.Plugin{
	Command:     []string{"hidetag", "tag"},
	Description: "Menciona a todos sin mostrar los @",
	Category:    "groups",
	IsGroup:     true,
	IsAdmin:     true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		groupMeta, err := client.GetGroupInfo(context.Background(), m.ChatJID)
		if err != nil {
			return m.Reply("No pude obtener la información del grupo.")
		}
		var mentions []string
		for _, p := range groupMeta.Participants {
			mentions = append(mentions, p.JID.String())
		}

		finalText := ctx.Text
		if finalText == "" && m.Quoted != nil {
			finalText = m.Quoted.Text
		}

		if finalText == "" && m.Quoted == nil && !m.IsMedia {
			return m.Reply("Ingresa un texto o responde a un mensaje para taguear.")
		}

		if m.Quoted != nil && m.Quoted.IsMedia {
			media, err := m.Quoted.Download()
			if err != nil {
				return m.Reply("Error al descargar el medio.")
			}
			mime := m.Quoted.Mimetype
			switch {
			case strings.Contains(mime, "image"):
				_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
					ImageMessage: &waProto.ImageMessage{
						Mimetype:      proto.String(mime),
						JPEGThumbnail: media,
						Caption:       proto.String(finalText),
						ContextInfo:   &waProto.ContextInfo{MentionedJID: mentions},
					},
				})
			case strings.Contains(mime, "video"):
				_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
					VideoMessage: &waProto.VideoMessage{
						Mimetype:    proto.String("video/mp4"),
						Caption:     proto.String(finalText),
						ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
					},
				})
			default:
				_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
					ExtendedTextMessage: &waProto.ExtendedTextMessage{
						Text:        proto.String(finalText),
						ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
					},
				})
			}
			return err
		}

		_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text:        proto.String(finalText),
				ContextInfo: &waProto.ContextInfo{MentionedJID: mentions},
			},
		})
		return err
	},
}

var kickPlugin = &plugin.Plugin{
	Command:     []string{"kick", "kill", "matar", "sacar"},
	Description: "Expulsa a un miembro del grupo",
	Category:    "groups",
	IsGroup:     true,
	IsAdmin:     true,
	IsBotAdmin:  true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		botJid := strings.Split(client.Store.ID.String(), ":")[0] + "@s.whatsapp.net"

		var user string
		if len(m.MentionedJID) > 0 {
			user = m.MentionedJID[0]
		} else if m.Quoted != nil {
			user = m.Quoted.Sender
		}
		if user == "" {
			return m.Reply("Etiqueta o responde al mensaje de la persona que quieres eliminar.")
		}

		targetUser := strings.Split(user, ":")[0] + "@s.whatsapp.net"
		if targetUser == botJid {
			return m.Reply("No puedo eliminarme a mí mismo del grupo.")
		}

		userJID, err := types.ParseJID(user)
		if err != nil {
			return m.Reply("JID inválido.")
		}
		_, err = client.UpdateGroupParticipants(context.Background(), m.ChatJID, []types.JID{userJID}, whatsmeow.ParticipantChangeRemove)
		if err != nil {
			return m.Reply("No se pudo eliminar al usuario. Verifica que sea un número válido o mis permisos.")
		}

		database := db.Get()
		chat := database.GetOrCreateChat(m.Chat)
		chat.Logs = append(chat.Logs, db.Log{Action: "kick", Target: targetUser, By: m.Sender, Date: time.Now().Format(time.RFC3339)})
		database.SaveChat(m.Chat, chat)

		return m.Reply("Usuario eliminado correctamente.")
	},
}

var promotePlugin = &plugin.Plugin{
	Command:     []string{"promote", "ascender", "daradmin"},
	Description: "Agrega un nuevo administrador",
	Category:    "groups",
	IsGroup:     true,
	IsAdmin:     true,
	IsBotAdmin:  true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		var target string
		if m.Quoted != nil {
			target = m.Quoted.Sender
		} else if len(m.MentionedJID) > 0 {
			target = m.MentionedJID[0]
		}
		if target == "" {
			return m.Reply("Etiqueta o responde al mensaje del usuario que deseas ascender.")
		}

		cleanTarget := strings.Split(target, ":")[0] + "@s.whatsapp.net"
		botJid := strings.Split(client.Store.ID.String(), ":")[0] + "@s.whatsapp.net"
		if cleanTarget == botJid {
			return m.Reply("Ya soy administrador del grupo.")
		}

		targetJID, err := types.ParseJID(target)
		if err != nil {
			return m.Reply("JID inválido.")
		}
		_, err = client.UpdateGroupParticipants(context.Background(), m.ChatJID, []types.JID{targetJID}, whatsmeow.ParticipantChangePromote)
		if err != nil {
			return m.Reply("No se pudo realizar el ascenso. Verifica mis permisos en el grupo.")
		}

		database := db.Get()
		chat := database.GetOrCreateChat(m.Chat)
		chat.Logs = append(chat.Logs, db.Log{Action: "promote", Option: "ADD ADMIN", Target: cleanTarget, By: m.Sender, Date: time.Now().Format(time.RFC3339)})
		database.SaveChat(m.Chat, chat)

		_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(fmt.Sprintf("*%s* ahora es administrador.", m.PushName)),
				ContextInfo: &waProto.ContextInfo{MentionedJID: []string{cleanTarget}},
			},
		})
		return err
	},
}

var demotePlugin = &plugin.Plugin{
	Command:     []string{"demote", "degradar", "quitaradmin"},
	Description: "Degrada a un administrador",
	Category:    "groups",
	IsGroup:     true,
	IsAdmin:     true,
	IsBotAdmin:  true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		var target string
		if m.Quoted != nil {
			target = m.Quoted.Sender
		} else if len(m.MentionedJID) > 0 {
			target = m.MentionedJID[0]
		}
		if target == "" {
			return m.Reply("Etiquete o responda al mensaje del admin que desea degradar.")
		}

		targetJID, err := types.ParseJID(target)
		if err != nil {
			return m.Reply("JID inválido.")
		}
		_, err = client.UpdateGroupParticipants(context.Background(), m.ChatJID, []types.JID{targetJID}, whatsmeow.ParticipantChangeDemote)
		if err != nil {
			return m.Reply("No se pudo degradar al usuario. Verifica que sea administrador.")
		}

		database := db.Get()
		chat := database.GetOrCreateChat(m.Chat)
		chat.Logs = append(chat.Logs, db.Log{Action: "demote", Option: "DEL ADMIN", By: m.Sender, Date: time.Now().Format(time.RFC3339)})
		database.SaveChat(m.Chat, chat)

		_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(fmt.Sprintf("*@%s* ha sido degradado de administrador.", strings.Split(target, "@")[0])),
				ContextInfo: &waProto.ContextInfo{MentionedJID: []string{target}},
			},
		})
		return err
	},
}

var onOffPlugin = &plugin.Plugin{
	Command:     []string{"on", "off"},
	Description: "Activa o desactiva funciones del grupo",
	Category:    "groups",
	IsGroup:     true,
	IsAdmin:     true,
	IsBotAdmin:  true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		isEnable := ctx.CommandName == "on"
		if len(ctx.Args) == 0 {
			return m.Reply("Debes especificar la *función*\n\n`Ejemplo`\n!on antilink\n!off antilink")
		}
		setting := strings.ToLower(ctx.Args[0])

		database := db.Get()
		chat := database.GetOrCreateChat(m.Chat)

		stateOn := func(s bool) string {
			if s {
				return "activada"
			}
			return "desactivada"
		}
		stateOnM := func(s bool) string {
			if s {
				return "activado"
			}
			return "desactivado"
		}

		switch setting {
		case "antilink":
			chat.Antilink = isEnable
			database.SaveChat(m.Chat, chat)
			return m.Reply(fmt.Sprintf("💬 La función *Antilink* ha sido *%s*", stateOn(isEnable)))
		case "adminonly", "onlyadmin":
			chat.Adminonly = isEnable
			database.SaveChat(m.Chat, chat)
			return m.Reply(fmt.Sprintf("💬 El *Modo Admins* ha sido *%s*", stateOnM(isEnable)))
		case "welcome":
			chat.Welcome = isEnable
			database.SaveChat(m.Chat, chat)
			return m.Reply(fmt.Sprintf("💬 La *Bienvenida* ha sido *%s*", stateOn(isEnable)))
		case "detect":
			chat.Detect = isEnable
			database.SaveChat(m.Chat, chat)
			return m.Reply(fmt.Sprintf("💬 El *Detector* ha sido *%s*", stateOnM(isEnable)))
		default:
			return m.Reply("Opción no *válida*\n\n- Opciones:\n`antilink`\n`welcome`\n`adminonly`\n`detect`\n\n> Ejemplo: .on welcome")
		}
	},
}

var changesPlugin = &plugin.Plugin{
	Command:     []string{"logs"},
	Description: "Muestra el historial de cambios de funciones del grupo",
	Category:    "groups",
	IsGroup:     true,
	IsAdmin:     true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		database := db.Get()
		chat := database.GetOrCreateChat(m.Chat)

		if len(chat.Logs) == 0 {
			return m.Reply("*No se registra historial de cambios en este grupo.*")
		}

		m.Reply("Generando reporte del historial, espere un momento...")

		text := "Logs\n\n> 𝖧𝗂𝗌𝗍𝗈𝗋𝗂𝖺𝗅 𝖽𝖾 𝗆𝗈𝖽𝗂𝖿𝗂𝖼𝖺𝖼𝗂𝗈𝗇𝖾𝗌 ›\n\n"
		var mentions []string
		for i, log := range chat.Logs {
			text += fmt.Sprintf("*Evento* › %d\n*Función* › %s\n*Acción* › %s\n*Usuario* › @%s\n*Fecha* › %s\n\n",
				i+1, log.Option, log.Action, strings.Split(log.By, "@")[0], log.Date)
			mentions = append(mentions, log.By)
		}

		_, err := client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(text),
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
