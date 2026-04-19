package dev

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"mini_wabot/lib/simple"
	"mini_wabot/plugin"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func Plugins() []*plugin.Plugin {
	return []*plugin.Plugin{backupPlugin, restartPlugin, updatePlugin}
}

var backupPlugin = &plugin.Plugin{
	Command:     []string{"backup", "respaldo", "creds"},
	Description: "Hace un respaldo de los archivos importantes",
	Category:    "developer",
	IsDev:       true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		credsPath := "./auth/creds.json"
		if _, err := os.Stat(credsPath); os.IsNotExist(err) {
			return m.Reply("⚠️ _El archivo *creds.json* no existe en el servidor._")
		}

		m.Reply("*☁️ Preparando envío de datos...*")

		creds, err := os.ReadFile(credsPath)
		if err != nil {
			return m.Reply("☁️ _Ocurrió un error al intentar leer o enviar las credenciales._")
		}

		senderJID, err := types.ParseJID(m.Sender)
		if err != nil {
			return m.Reply("JID inválido.")
		}

		resp, err := client.Upload(context.Background(), creds, whatsmeow.MediaDocument)
		if err != nil {
			return m.Reply("☁️ _Ocurrió un error al intentar enviar las credenciales._")
		}

		_, err = client.SendMessage(context.Background(), senderJID, &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				URL:           proto.String(resp.URL),
				DirectPath:    proto.String(resp.DirectPath),
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(creds))),
				Mimetype:      proto.String("application/json"),
				FileName:      proto.String("creds.json"),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      proto.String(m.ID),
					Participant:   proto.String(m.Sender),
					QuotedMessage: m.Msg,
				},
			},
		})
		if err != nil {
			return m.Reply("☁️ _Ocurrió un error al intentar enviar las credenciales._")
		}
		return m.Reply("✅ _Respaldo enviado a tu chat privado._")
	},
}

var restartPlugin = &plugin.Plugin{
	Command:     []string{"restart", "reiniciar"},
	Description: "Reinicia el bot remotamente",
	Category:    "developer",
	IsDev:       true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		_, err := client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String("☁️ *Lurus* se estará reiniciando...\n> Espere mientras el sock se reinicia"),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:      proto.String(m.ID),
					Participant:   proto.String(m.Sender),
					QuotedMessage: m.Msg,
				},
			},
		})
		if err != nil {
			return err
		}
		time.AfterFunc(3*time.Second, func() { os.Exit(0) })
		return nil
	},
}

var updatePlugin = &plugin.Plugin{
	Command:     []string{"update", "actualizar"},
	Description: "Actualiza el repositorio",
	Category:    "owner",
	IsOwner:     true,
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		m.Reply("Buscando actualizaciones en el repositorio...")
		out, err := exec.Command("git", "pull").CombinedOutput()
		if err != nil {
			return m.Reply(fmt.Sprintf("Error durante la actualización:\n%s", err.Error()))
		}
		output := string(out)
		if output == "Already up to date.\n" || output == "Already up to date.\r\n" {
			return m.Reply("El sistema ya esta actualizado.")
		}
		return m.Reply("Actualización completada con exito.\n\n" + output)
	},
}
