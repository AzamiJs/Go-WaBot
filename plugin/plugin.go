package plugin

import (
	"mini_wabot/lib/simple"

	"go.mau.fi/whatsmeow"
)

type CommandContext struct {
	Args        []string
	Prefix      string
	CommandName string
	IsAdmins    bool
	IsBotAdmin  bool
	SenderJid   string
	Text        string
}

type Plugin struct {
	Command     []string
	Description string
	Category    string
	IsOwner     bool
	IsDev       bool
	IsGroup     bool
	IsAdmin     bool
	IsBotAdmin  bool
	Run         func(client *whatsmeow.Client, m *simple.Message, ctx CommandContext) error
}
