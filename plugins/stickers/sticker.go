package stickers

import (
	"context"
	"strings"

	"mini_wabot/lib/simple"
	stk "mini_wabot/lib/stickers"
	"mini_wabot/plugin"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

func Plugins() []*plugin.Plugin {
	return []*plugin.Plugin{stickerPlugin}
}

var stickerPlugin = &plugin.Plugin{
	Command:     []string{"sticker", "s"},
	Description: "Crea un sticker a partir de una imagen o video",
	Category:    "stickers",
	Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
		var mime string
		var isQuoted bool

		if m.Quoted != nil && m.Quoted.IsMedia {
			mime = m.Quoted.Mimetype
			isQuoted = true
		} else if m.IsMedia {
			mime = m.Mimetype
		}

		if !strings.Contains(mime, "image") && !strings.Contains(mime, "video") {
			return m.Reply("Responde a una imagen o video para crear el sticker.")
		}

		m.Reply("Procesando su sticker...")

		var mediaBuffer []byte
		var err error
		if isQuoted {
			mediaBuffer, err = m.Quoted.Download()
		} else {
			mediaBuffer, err = m.Download()
		}
		if err != nil || len(mediaBuffer) == 0 {
			return m.Reply("No pude descargar el medio.")
		}

		meta := stk.StickerMeta{Packname: "Lurus", Author: "Zam"}
		var stickerBuf []byte
		if strings.Contains(mime, "image") {
			stickerBuf, err = stk.WriteExifImg(mediaBuffer, meta)
		} else {
			stickerBuf, err = stk.WriteExifVid(mediaBuffer, meta)
		}
		if err != nil || len(stickerBuf) == 0 {
			return m.Reply("Ocurrió un error al crear el sticker. Revisa los logs del panel.")
		}

		uploaded, err := client.Upload(context.Background(), stickerBuf, whatsmeow.MediaImage)
		if err != nil {
			return m.Reply("Error al subir el sticker.")
		}

		_, err = client.SendMessage(context.Background(), m.ChatJID, &waProto.Message{
			StickerMessage: &waProto.StickerMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(stickerBuf))),
				Mimetype:      proto.String("image/webp"),
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
