package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"mini_wabot/config"
	"mini_wabot/lib/db"
	"mini_wabot/lib/simple"
	"mini_wabot/lib/utils"
	"mini_wabot/plugin"
	devplugins "mini_wabot/plugins/dev"
	generalplugins "mini_wabot/plugins/general"
	groupplugins "mini_wabot/plugins/groups"
	ownerplugins "mini_wabot/plugins/owner"
	stickerplugins "mini_wabot/plugins/stickers"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

var (
	commandIndex = make(map[string]*plugin.Plugin)
	indexMu      sync.RWMutex
)

func register(p *plugin.Plugin) {
	indexMu.Lock()
	defer indexMu.Unlock()
	for _, alias := range p.Command {
		commandIndex[alias] = p
	}
}

func LoadPlugins() {
	for _, p := range generalplugins.Plugins() {
		register(p)
	}
	for _, p := range groupplugins.Plugins() {
		register(p)
	}
	for _, p := range devplugins.Plugins() {
		register(p)
	}
	for _, p := range ownerplugins.Plugins() {
		register(p)
	}
	for _, p := range stickerplugins.Plugins() {
		register(p)
	}
	fmt.Printf("%d plugins cargados en caché.\n", len(commandIndex))
}

var ownerSet = func() map[string]bool {
	s := make(map[string]bool)
	for _, n := range config.Owners {
		s[n+"@s.whatsapp.net"] = true
	}
	return s
}()

var devSet = func() map[string]bool {
	s := make(map[string]bool)
	for _, n := range config.Devs {
		s[n+"@s.whatsapp.net"] = true
	}
	return s
}()

type groupCacheEntry struct {
	data *types.GroupInfo
	ts   int64
}

var (
	groupCache   = make(map[string]groupCacheEntry)
	groupCacheMu sync.RWMutex
)

func cachedGroupMeta(client *whatsmeow.Client, chatId string) *types.GroupInfo {
	groupCacheMu.RLock()
	cached, ok := groupCache[chatId]
	groupCacheMu.RUnlock()
	if ok && time.Now().UnixMilli()-cached.ts < 5*60*1000 {
		return cached.data
	}
	jid, err := types.ParseJID(chatId)
	if err != nil {
		return nil
	}
	info, err := client.GetGroupInfo(context.Background(), jid)
	if err != nil {
		return nil
	}
	groupCacheMu.Lock()
	groupCache[chatId] = groupCacheEntry{data: info, ts: time.Now().UnixMilli()}
	groupCacheMu.Unlock()
	return info
}

func Handle(client *whatsmeow.Client, m *simple.Message) {
	handleAsync(client, m)
}

func handleAsync(client *whatsmeow.Client, m *simple.Message) {
	if m.Msg == nil {
		return
	}

	prefix := ""
	for _, p := range config.Prefix {
		if strings.HasPrefix(m.Text, p) {
			prefix = p
			break
		}
	}

	if prefix == "" {
		AntiLink(client, m)
		return
	}

	body := strings.TrimSpace(m.Text[len(prefix):])
	parts := strings.Fields(body)
	if len(parts) == 0 {
		return
	}
	commandName := strings.ToLower(parts[0])
	args := parts[1:]
	text := strings.Join(args, " ")

	indexMu.RLock()
	cmd, found := commandIndex[commandName]
	indexMu.RUnlock()
	if !found {
		return
	}

	database := db.Get()
	senderPhone := strings.Split(m.Sender, "@")[0]

	database.GetOrCreateUser(senderPhone)
	database.GetOrCreateChat(m.Chat)

	var groupMetadata *types.GroupInfo
	var isAdmins, isBotAdmin bool

	if m.IsGroup && (cmd.IsAdmin || cmd.IsBotAdmin || cmd.IsGroup) {
		groupMetadata = cachedGroupMeta(client, m.Chat)
		if groupMetadata != nil {
			botRaw := strings.Split(strings.Split(client.Store.ID.String(), ":")[0], "@")[0]
			senderRaw := strings.Split(strings.Split(m.Sender, ":")[0], "@")[0]
			for _, p := range groupMetadata.Participants {
				pid := strings.Split(strings.Split(p.JID.String(), ":")[0], "@")[0]
				isAdmin := p.IsAdmin || p.IsSuperAdmin
				if pid == botRaw && isAdmin {
					isBotAdmin = true
				}
				if pid == senderRaw && isAdmin {
					isAdmins = true
				}
			}
		}
	}

	senderJid := utils.ResolveLidToRealJid(m.Sender, groupMetadata)

	now := time.Now().In(mustLoadLocation("America/Mexico_City")).Format("15:04:05")
	fmt.Println("╭───────────────────────────────────────╮")
	fmt.Printf("│ CMD : %s\n", commandName)
	fmt.Printf("│ USER: %s\n", m.PushName)
	fmt.Printf("│ TIME: %s\n", now)
	fmt.Println("╰───────────────────────────────────────╯")

	if cmd.IsOwner && !ownerSet[senderJid] {
		m.Reply("Dueño solamente.")
		return
	}
	if cmd.IsDev && !devSet[senderJid] {
		m.Reply("Creador solamente.")
		return
	}
	if cmd.IsGroup && !m.IsGroup {
		m.Reply("Grupos solamente.")
		return
	}
	if cmd.IsAdmin && !isAdmins {
		m.Reply("Admins solamente.")
		return
	}
	if cmd.IsBotAdmin && !isBotAdmin {
		m.Reply("Hazme admin primero.")
		return
	}

	ctx := plugin.CommandContext{
		Args:        args,
		Prefix:      prefix,
		CommandName: commandName,
		IsAdmins:    isAdmins,
		IsBotAdmin:  isBotAdmin,
		SenderJid:   senderJid,
		Text:        text,
	}

	if err := cmd.Run(client, m, ctx); err != nil {
		fmt.Printf("Error en plugin '%s': %v\n", commandName, err)
		m.Reply("_Ocurrió un error al ejecutar el comando. Intenta de nuevo._")
	}
}

var mexicoTZ = func() *time.Location {
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func mustLoadLocation(name string) *time.Location {
	if name == "America/Mexico_City" {
		return mexicoTZ
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}
