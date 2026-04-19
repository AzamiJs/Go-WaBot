package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type UserData struct {
	User  string `json:"user"`
	Coins int    `json:"coins"`
	Name  string `json:"name,omitempty"`
}

type ChatData struct {
	ID        string `json:"id"`
	Detect    bool   `json:"detect"`
	Welcome   bool   `json:"welcome"`
	Antilink  bool   `json:"antilink"`
	Adminonly bool   `json:"adminonly,omitempty"`
	Logs      []Log  `json:"logs,omitempty"`
}

type Log struct {
	Action  string `json:"action"`
	Option  string `json:"option,omitempty"`
	Target  string `json:"target,omitempty"`
	By      string `json:"by"`
	Date    string `json:"date"`
}

type Token struct {
	Token     string  `json:"token"`
	UsesLeft  int     `json:"uses_left"`
	ExpiresAt int64   `json:"expires_at"`
	CreatedBy string  `json:"created_by"`
	UsedBy    *string `json:"used_by"`
}

type DB struct {
	mu       sync.RWMutex
	sql      *sql.DB
	Users    map[string]*UserData
	Chats    map[string]*ChatData
	Settings map[string]map[string]any
}

var instance *DB

func Load() *DB {
	sqlDB, err := sql.Open("sqlite3", "./database.db?_timeout=10000&_journal=WAL")
	if err != nil {
		log.Fatalf("Error abriendo SQLite: %v", err)
	}

	_, err = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, data TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS chats (id TEXT PRIMARY KEY, data TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS settings (id TEXT PRIMARY KEY, data TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS tokens (
			token TEXT PRIMARY KEY,
			uses_left INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			created_by TEXT NOT NULL,
			used_by TEXT
		);
	`)
	if err != nil {
		log.Fatalf("Error creando tablas: %v", err)
	}

	d := &DB{
		sql:      sqlDB,
		Users:    make(map[string]*UserData),
		Chats:    make(map[string]*ChatData),
		Settings: make(map[string]map[string]any),
	}

	rows, _ := sqlDB.Query("SELECT id, data FROM users")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, data string
			rows.Scan(&id, &data)
			var u UserData
			json.Unmarshal([]byte(data), &u)
			d.Users[id] = &u
		}
	}

	rows2, _ := sqlDB.Query("SELECT id, data FROM chats")
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var id, data string
			rows2.Scan(&id, &data)
			var c ChatData
			json.Unmarshal([]byte(data), &c)
			d.Chats[id] = &c
		}
	}

	instance = d
	fmt.Println("SQLite: Base de datos cargada y sincronizada.")
	return d
}

func Get() *DB {
	return instance
}

func (d *DB) SaveUser(id string, u *UserData) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Users[id] = u
	b, _ := json.Marshal(u)
	d.sql.Exec("INSERT OR REPLACE INTO users (id, data) VALUES (?, ?)", id, string(b))
}

func (d *DB) SaveChat(id string, c *ChatData) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Chats[id] = c
	b, _ := json.Marshal(c)
	d.sql.Exec("INSERT OR REPLACE INTO chats (id, data) VALUES (?, ?)", id, string(b))
}

func (d *DB) GetOrCreateUser(id string) *UserData {
	d.mu.RLock()
	u, ok := d.Users[id]
	d.mu.RUnlock()
	if ok {
		return u
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if u, ok := d.Users[id]; ok {
		return u
	}
	u = &UserData{User: id, Coins: 0}
	d.Users[id] = u
	b, _ := json.Marshal(u)
	d.sql.Exec("INSERT OR REPLACE INTO users (id, data) VALUES (?, ?)", id, string(b))
	return u
}

func (d *DB) GetOrCreateChat(id string) *ChatData {
	d.mu.RLock()
	c, ok := d.Chats[id]
	d.mu.RUnlock()
	if ok {
		return c
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if c, ok := d.Chats[id]; ok {
		return c
	}
	c = &ChatData{ID: id, Detect: true, Welcome: true, Antilink: true}
	d.Chats[id] = c
	b, _ := json.Marshal(c)
	d.sql.Exec("INSERT OR REPLACE INTO chats (id, data) VALUES (?, ?)", id, string(b))
	return c
}

func (d *DB) DeleteUser(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.Users, id)
	d.sql.Exec("DELETE FROM users WHERE id = ?", id)
}

func (d *DB) DeleteChat(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.Chats, id)
	d.sql.Exec("DELETE FROM chats WHERE id = ?", id)
}

func (d *DB) TokenCreate(token string, usesLeft int, expiresAt int64, createdBy string) {
	d.sql.Exec(
		"INSERT OR REPLACE INTO tokens (token, uses_left, expires_at, created_by, used_by) VALUES (?, ?, ?, ?, NULL)",
		token, usesLeft, expiresAt, createdBy,
	)
}

func (d *DB) TokenGet(token string) *Token {
	row := d.sql.QueryRow("SELECT token, uses_left, expires_at, created_by, used_by FROM tokens WHERE token = ?", token)
	var t Token
	var usedBy sql.NullString
	err := row.Scan(&t.Token, &t.UsesLeft, &t.ExpiresAt, &t.CreatedBy, &usedBy)
	if err != nil {
		return nil
	}
	if usedBy.Valid {
		t.UsedBy = &usedBy.String
	}
	return &t
}

func (d *DB) TokenDelete(token string) {
	d.sql.Exec("DELETE FROM tokens WHERE token = ?", token)
}

func (d *DB) TokenList() []Token {
	rows, err := d.sql.Query("SELECT token, uses_left, expires_at, created_by, used_by FROM tokens")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var tokens []Token
	for rows.Next() {
		var t Token
		var usedBy sql.NullString
		rows.Scan(&t.Token, &t.UsesLeft, &t.ExpiresAt, &t.CreatedBy, &usedBy)
		if usedBy.Valid {
			t.UsedBy = &usedBy.String
		}
		tokens = append(tokens, t)
	}
	return tokens
}

func (d *DB) RLock()   { d.mu.RLock() }
func (d *DB) RUnlock() { d.mu.RUnlock() }
