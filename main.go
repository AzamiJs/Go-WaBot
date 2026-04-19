package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mini_wabot/handler"
	"mini_wabot/lib/db"
	"mini_wabot/lib/simple"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

const (
	authFolder  = "auth"
	maxRetries  = 5
	cacheMaxMB  = 100
)

var (
	retryCount = 0
)

func box(lines ...string) {
	fmt.Println("╭───────────────────────────────────────╮")
	for _, l := range lines {
		fmt.Printf("│  %s\n", l)
	}
	fmt.Println("╰───────────────────────────────────────╯")
}

func clearSession() {
	if _, err := os.Stat(authFolder); err == nil {
		os.RemoveAll(authFolder)
		box("⚠  Sesión eliminada automáticamente", "Reinicia para vincular de nuevo")
	}
}

func cleanCache() {
	tmpFolder := "./tmp"
	if _, err := os.Stat(tmpFolder); err == nil {
		entries, _ := os.ReadDir(tmpFolder)
		cleaned := 0
		for _, e := range entries {
			os.Remove(tmpFolder + "/" + e.Name())
			cleaned++
		}
		if cleaned > 0 {
			fmt.Printf("[ 🧹 ] Cache: %d archivos eliminados\n", cleaned)
		}
	}
}

func question(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

type groupMetaCacheEntry struct {
	data *types.GroupInfo
	ts   int64
}

var groupMetaCache = make(map[string]groupMetaCacheEntry)

func getGroupMeta(client *whatsmeow.Client, groupId string) *types.GroupInfo {
	if cached, ok := groupMetaCache[groupId]; ok {
		if time.Now().UnixMilli()-cached.ts < 5*60*1000 {
			return cached.data
		}
	}
	jid, err := types.ParseJID(groupId)
	if err != nil {
		return nil
	}
	info, err := client.GetGroupInfo(context.Background(), jid)
	if err != nil {
		return nil
	}
	groupMetaCache[groupId] = groupMetaCacheEntry{data: info, ts: time.Now().UnixMilli()}
	return info
}

func startBot() {
	database := db.Load()
	_ = database

	handler.LoadPlugins()

	cleanCache()
	go func() {
		for range time.Tick(6 * time.Hour) {
			cleanCache()
		}
	}()

	os.MkdirAll(authFolder, 0755)

	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+authFolder+"/session.db?_foreign_keys=on", dbLog)
	if err != nil {
		fmt.Println("Error abriendo store:", err)
		return
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		fmt.Println("Error obteniendo dispositivo:", err)
		return
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	hasSession := client.Store.ID != nil

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			handleMessage(client, v)
		case *events.GroupInfo:
			for _, memberJID := range v.Join {
				handleGroupAdd(client, v.JID.String(), memberJID, true)
			}
			for _, memberJID := range v.Leave {
				handleGroupAdd(client, v.JID.String(), memberJID, false)
			}
		case *events.Connected:
			retryCount = 0
			box("✦  BOT CONECTADO EXITOSAMENTE  ✦")
		case *events.Disconnected:
			handleDisconnect(client)
		}
	})

	if err := client.Connect(); err != nil {
		fmt.Println("Error conectando:", err)
		return
	}

	if !hasSession {
		box("✦  LURUS  —  VINCULAR SESIÓN  ✦")
		phone := question("\nNúmero (código de país, sin +)\n  › ")
		phone = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, phone)
		phone = strings.TrimPrefix(phone, "+")
		if len(phone) < 7 {
			fmt.Println("\n  ✗ Número inválido.\n")
			client.Disconnect()
			return
		}
		code, err := client.PairPhone(context.Background(), phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		if err != nil {
			fmt.Println("\n  ✗ Error al generar el código:", err)
			client.Disconnect()
			return
		}
		formatted := code
		if len(code) == 8 {
			formatted = code[:4] + "-" + code[4:]
		} else {
			parts := []string{}
			for i := 0; i < len(code); i += 4 {
				end := i + 4
				if end > len(code) {
					end = len(code)
				}
				parts = append(parts, code[i:end])
			}
			formatted = strings.Join(parts, "-")
		}
		box(
			"CÓDIGO › "+formatted,
			"WhatsApp → Dispositivos vinculados",
			"→ Vincular con número de teléfono",
		)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	client.Disconnect()
}

func handleMessage(client *whatsmeow.Client, evt *events.Message) {
	if evt.Message == nil {
		return
	}
	if evt.Info.Chat.String() == "status@broadcast" {
		return
	}
	m := simple.BuildMessage(client, evt)
	handler.Handle(client, m)
}

func handleGroupAdd(client *whatsmeow.Client, groupId string, memberJID types.JID, isAdd bool) {
	database := db.Get()
	chat := database.GetOrCreateChat(groupId)
	metadata := getGroupMeta(client, groupId)
	if metadata == nil {
		return
	}

	jid := memberJID.String()
	phone := strings.Split(jid, "@")[0]
	pp := getProfilePic(client, jid)

	if isAdd && chat.Welcome {
		text := fmt.Sprintf("> Welcome *@%s* to *%s* 👋", phone, metadata.Name)
		client.SendMessage(context.Background(), metadata.JID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(text),
				ContextInfo: &waProto.ContextInfo{
					MentionedJID: []string{jid},
					ExternalAdReply: &waProto.ContextInfo_ExternalAdReplyInfo{
						Title:                 proto.String("L U R U S"),
						Body:                  proto.String(metadata.Name),
						MediaType:             waProto.ContextInfo_ExternalAdReplyInfo_IMAGE.Enum(),
						RenderLargerThumbnail: proto.Bool(true),
						ThumbnailURL:          proto.String(pp),
						SourceURL:             proto.String("github.com/AzamiJs"),
					},
				},
			},
		})
	} else if !isAdd && chat.Welcome {
		text := fmt.Sprintf("> Adiós, *@%s* ha salido del grupo *%s* 👋", phone, metadata.Name)
		client.SendMessage(context.Background(), metadata.JID, &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(text),
				ContextInfo: &waProto.ContextInfo{
					MentionedJID: []string{jid},
					ExternalAdReply: &waProto.ContextInfo_ExternalAdReplyInfo{
						Title:                 proto.String("L U R U S"),
						Body:                  proto.String(metadata.Name),
						MediaType:             waProto.ContextInfo_ExternalAdReplyInfo_IMAGE.Enum(),
						RenderLargerThumbnail: proto.Bool(true),
						ThumbnailURL:          proto.String(pp),
						SourceURL:             proto.String("github.com/AzamiJs"),
					},
				},
			},
		})
	}
}

func getProfilePic(client *whatsmeow.Client, jid string) string {
	parsed, err := types.ParseJID(jid)
	if err != nil {
		return "https://stellarwa.xyz/files/1755559736781.jpeg"
	}
	info, err := client.GetProfilePictureInfo(context.Background(), parsed, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err != nil || info == nil {
		return "https://stellarwa.xyz/files/1755559736781.jpeg"
	}
	return info.URL
}

func handleDisconnect(client *whatsmeow.Client) {
	retryCount++
	if retryCount >= maxRetries {
		box(
			fmt.Sprintf("✗  %d intentos fallidos", maxRetries),
			"Sesión corrupta — eliminando...",
		)
		retryCount = 0
		clearSession()
		time.AfterFunc(3*time.Second, startBot)
		return
	}
	delay := time.Duration(5*retryCount) * time.Second
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	fmt.Printf("[ ↻ ] Reintentando en %ds... (%d/%d)\n", int(delay.Seconds()), retryCount, maxRetries)
	time.AfterFunc(delay, func() {
		if err := client.Connect(); err != nil {
			handleDisconnect(client)
		}
	})
}

func main() {
	startBot()
}
