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

var (
    instance   *DB
    instanceMu sync.Mutex
)

func Load() *DB {
    instanceMu.Lock()
    if instance != nil && instance.sql != nil {
        db := instance
        instanceMu.Unlock()
        return db
    }
    instanceMu.Unlock()

    sqlDB, err := sql.Open("sqlite3", "./database.db?_timeout=10000&_journal=WAL")
    if err != nil {
        log.Fatalf("Error abriendo SQLite: %v", err)
    }

    if _, err = sqlDB.Exec(`
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
    `); err != nil {
        _ = sqlDB.Close()
        log.Fatalf("Error creando tablas: %v", err)
    }

    d := &DB{
        sql:      sqlDB,
        Users:    make(map[string]*UserData),
        Chats:    make(map[string]*ChatData),
        Settings: make(map[string]map[string]any),
    }

    if err := d.loadUsers(); err != nil {
        log.Printf("Advertencia: no se pudieron cargar usuarios: %v", err)
    }
    if err := d.loadChats(); err != nil {
        log.Printf("Advertencia: no se pudieron cargar chats: %v", err)
    }

    instanceMu.Lock()
    instance = d
    instanceMu.Unlock()

    log.Println("SQLite: Base de datos cargada y sincronizada.")
    return d
}

func Get() *DB {
    instanceMu.Lock()
    if instance != nil && instance.sql != nil {
        db := instance
        instanceMu.Unlock()
        return db
    }
    instanceMu.Unlock()
    return Load()
}

func (d *DB) Close() {
    if d == nil || d.sql == nil {
        return
    }
    _ = d.sql.Close()
    instanceMu.Lock()
    if instance == d {
        instance = nil
    }
    instanceMu.Unlock()
}

func (d *DB) loadUsers() error {
    rows, err := d.sql.Query("SELECT id, data FROM users")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id, data string
        if err := rows.Scan(&id, &data); err != nil {
            return err
        }
        var u UserData
        if err := json.Unmarshal([]byte(data), &u); err != nil {
            return err
        }
        d.Users[id] = &u
    }
    return rows.Err()
}

func (d *DB) loadChats() error {
    rows, err := d.sql.Query("SELECT id, data FROM chats")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id, data string
        if err := rows.Scan(&id, &data); err != nil {
            return err
        }
        var c ChatData
        if err := json.Unmarshal([]byte(data), &c); err != nil {
            return err
        }
        d.Chats[id] = &c
    }
    return rows.Err()
}

func (d *DB) persist(table, id string, value any) {
    b, err := json.Marshal(value)
    if err != nil {
        log.Printf("db persist marshal failed for %s/%s: %v", table, id, err)
        return
    }
    if _, err = d.sql.Exec(fmt.Sprintf("INSERT OR REPLACE INTO %s (id, data) VALUES (?, ?)", table), id, string(b)); err != nil {
        log.Printf("db persist failed for %s/%s: %v", table, id, err)
    }
}

func (d *DB) SaveUser(id string, u *UserData) {
    d.mu.Lock()
    d.Users[id] = u
    d.mu.Unlock()
    d.persist("users", id, u)
}

func (d *DB) SaveChat(id string, c *ChatData) {
    d.mu.Lock()
    d.Chats[id] = c
    d.mu.Unlock()
    d.persist("chats", id, c)
}

func (d *DB) GetOrCreateUser(id string) *UserData {
    d.mu.RLock()
    u, ok := d.Users[id]
    d.mu.RUnlock()
    if ok {
        return u
    }

    d.mu.Lock()
    if u, ok = d.Users[id]; ok {
        d.mu.Unlock()
        return u
    }
    u = &UserData{User: id, Coins: 0}
    d.Users[id] = u
    d.mu.Unlock()
    d.persist("users", id, u)
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
    if c, ok = d.Chats[id]; ok {
        d.mu.Unlock()
        return c
    }
    c = &ChatData{ID: id, Detect: true, Welcome: true, Antilink: true}
    d.Chats[id] = c
    d.mu.Unlock()
    d.persist("chats", id, c)
    return c
}

func (d *DB) DeleteUser(id string) {
    d.mu.Lock()
    delete(d.Users, id)
    d.mu.Unlock()
    d.sql.Exec("DELETE FROM users WHERE id = ?", id)
}

func (d *DB) DeleteChat(id string) {
    d.mu.Lock()
    delete(d.Chats, id)
    d.mu.Unlock()
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
        if err := rows.Scan(&t.Token, &t.UsesLeft, &t.ExpiresAt, &t.CreatedBy, &usedBy); err != nil {
            continue
        }
        if usedBy.Valid {
            t.UsedBy = &usedBy.String
        }
        tokens = append(tokens, t)
    }
    return tokens
}

func (d *DB) RLock()   { d.mu.RLock() }
func (d *DB) RUnlock() { d.mu.RUnlock() }
