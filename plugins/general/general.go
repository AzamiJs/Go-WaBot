package general

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"mini_wabot/plugin"
	"mini_wabot/lib/simple"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

var mexicoTZ = func() *time.Location {
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		return time.UTC
	}
	return loc
}()

var startTime = time.Now()

func Plugins() []*plugin.Plugin {
	return []*plugin.Plugin{pingPlugin, infoPlugin, menuPlugin, testPlugin}
}

var pingPlugin = &plugin.Plugin{
	Command:     []string{"ping"},
	Description: "Chequea la conexión y muestra info del bot",
	Category:    "general",
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		start := time.Now()

		up := time.Since(startTime)
		h := int(up.Hours())
		min := int(up.Minutes()) % 60
		s := int(up.Seconds()) % 60

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		ramMB := float64(memStats.HeapInuse) / 1024 / 1024

		sender := strings.Split(m.Sender, "@")[0]
		latency := float64(time.Since(start).Microseconds()) / 1000.0

		msg := fmt.Sprintf("Hola, %s ☀️\n\n`Ping:` %.2f ms\n`Uptime:` [ %dh %dm %ds ]\n`RAM usada:` %.2f MB\n`Usuario ID:` @%s",
			m.PushName, latency, h, min, s, ramMB, sender)

		_, err := client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(msg),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      proto.String(m.ID),
					Participant:   proto.String(m.Sender),
					QuotedMessage: m.Msg,
					MentionedJID:  []string{m.Sender},
				},
			},
		})
		return err
	},
}

var infoPlugin = &plugin.Plugin{
	Command:     []string{"info", "botinfo"},
	Description: "Muestra los datos técnicos del bot",
	Category:    "general",
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		up := time.Since(startTime)
		h := int(up.Hours())
		min := int(up.Minutes()) % 60
		s := int(up.Seconds()) % 60

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		freeMem := float64(memStats.Sys-memStats.HeapInuse) / 1024 / 1024
		totalMem := float64(memStats.Sys) / 1024 / 1024

		now := time.Now().In(mexicoTZ).Format("01/02/2006, 15:04:05")
		banner := "https://cdn.evogb.org/AzamiJs/nlqIX-ecaaf4d537558d8dd7122798523ea667.jpg"

		info := fmt.Sprintf("*Nombre Bot:* Ai Lurus\n*Versión:* Go@1.0.0\n*Autor:* Zam | Ai Lurus\n*Uptime:* %dh %dm %ds\n*Plataforma:* %s %s\n*Go:* %s\n\n*Memoria:* %.0f MiB libre / %.0f MiB total\n\n*Fecha & Hora:* %s",
			h, min, s, runtime.GOOS, runtime.GOARCH, runtime.Version(), freeMem, totalMem, now)

		_, err := client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(strings.TrimSpace(info)),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      proto.String(m.ID),
					Participant:   proto.String(m.Sender),
					QuotedMessage: m.Msg,
					ExternalAdReply: &waProto.ContextInfo_ExternalAdReplyInfo{
						Title:                 proto.String("Ai Lurus"),
						Body:                  proto.String(fmt.Sprintf("Hora: %s | Zam", now)),
						MediaType:             waProto.ContextInfo_ExternalAdReplyInfo_IMAGE.Enum(),
						RenderLargerThumbnail: proto.Bool(false),
						ThumbnailURL:          proto.String(banner),
						SourceURL:             proto.String("https://whatsapp.com/channel/0029Vb5vOO0ADTOGbo78x03q"),
					},
				},
			},
		})
		return err
	},
}

var menuPlugin = &plugin.Plugin{
	Command:  []string{"menu", "help"},
	Category: "general",
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		now := time.Now().In(mexicoTZ).Format("15:04:05")
		name := m.PushName
		if name == "" {
			name = "User"
		}

		menuText := fmt.Sprintf("*¡Hola %s!* espero que te encuentres bien, aquí tienes mi lista de funciones... *!!* ˙\n\n", name)

		categories := map[string][]string{
			"general":  {"ping", "info", "menu"},
			"groups":   {"hidetag", "kick", "promote", "demote", "on", "off", "logs"},
			"stickers": {"sticker"},
			"owner":    {"listusers", "update"},
			"dev":      {"backup", "restart"},
		}
		order := []string{"dev", "general", "groups", "owner", "stickers"}

		for _, cat := range order {
			cmds := categories[cat]
			menuText += fmt.Sprintf("\n☁️ `%s:`\n", strings.ToUpper(cat))
			for _, cmd := range cmds {
				menuText += fmt.Sprintf("→ %s%s\n", ctx.Prefix, cmd)
			}
		}
		menuText += "`Canal` https://whatsapp.com/channel/0029Vb5vOO0ADTOGbo78x03q\n"

		_, err := client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(strings.TrimSpace(menuText)),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      proto.String(m.ID),
					Participant:   proto.String(m.Sender),
					QuotedMessage: m.Msg,
					ExternalAdReply: &waProto.ContextInfo_ExternalAdReplyInfo{
						Title:                 proto.String("Ai Lurus"),
						Body:                  proto.String(fmt.Sprintf("Hora: %s | Zam", now)),
						MediaType:             waProto.ContextInfo_ExternalAdReplyInfo_IMAGE.Enum(),
						RenderLargerThumbnail: proto.Bool(false),
						ThumbnailURL:          proto.String("https://cdn.evogb.org/AzamiJs/NevMd-37f1e73bea2c8868c8a2a1b288f42b67.jpg"),
						SourceURL:             proto.String("https://whatsapp.com/channel/0029VamN7Jt30LKWJlyWNB0D"),
					},
				},
			},
		})
		return err
	},
}

var testPlugin = &plugin.Plugin{
	Command:     []string{"test"},
	Description: "Mide la latencia real de red con WhatsApp",
	Category:    "general",
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		start := time.Now()

		_, err := client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			Conversation: proto.String("⏱️ Calculando ping de red..."),
		})
		if err != nil {
			return err
		}

		latency := time.Since(start).Milliseconds()

		msg := fmt.Sprintf("📡 *Test de Red*\n\n`Ping red:` %d ms\n`Ping código:` < 1 ms\n\n> El ping de red depende del servidor, no del código.", latency)

		_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(msg),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      proto.String(m.ID),
					Participant:   proto.String(m.Sender),
					QuotedMessage: m.Msg,
				},
			},
		})
		return err
	},
}
