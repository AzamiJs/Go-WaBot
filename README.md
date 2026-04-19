# Mini WaBot — Go

> Bot de WhatsApp escrito en Go, transcripción de [AzamiJs/Mini_WaBot](https://github.com/AzamiJs/Mini_WaBot) en TypeScript.

<p align="center">
  <img src="https://avatars.githubusercontent.com/AzamiJs" width="220px"/>
</p>

## Requisitos

- Go 1.21+
- GCC (para go-sqlite3, que usa CGO)
- ffmpeg (para el comando sticker)
- python3 (para inyectar EXIF en stickers)

## Instalación

```bash
# Instalar dependencias
go mod tidy

# Compilar
go build -o bot .

# Ejecutar
./bot
```

## Estructura

```
├── main.go                   # Punto de entrada, conexión y eventos
├── config/config.go          # Owners, devs, prefijos
├── plugin/plugin.go          # Tipo Plugin compartido
├── handler/
│   ├── handler.go            # Despachador de comandos
│   └── antilink.go           # Sistema antilink
├── lib/
│   ├── db/data.go            # SQLite + caché en memoria
│   ├── simple/simple.go      # Wrapper de mensajes (smsg)
│   ├── utils/utils.go        # Resolución de LID → JID
│   └── stickers/stickers.go  # Conversión WebP + EXIF
└── plugins/
    ├── general/              # ping, info, menu
    ├── groups/               # hidetag, kick, promote, demote, on/off, logs
    ├── dev/                  # backup, restart, update
    ├── owner/                # listusers
    └── stickers/             # sticker
```

## Vinculación

_Al iniciar sin sesión, ingresa tu número (con código de país, sin +) y recibirás un código de emparejamiento en la terminal para ingresar en WhatsApp → Dispositivos vinculados → Vincular con código de teléfono._

## Agregar plugins

Crea un archivo en la carpeta de categoría correspondiente, define un `*plugin.Plugin` y agrégalo al slice en `Plugins()` del mismo paquete.

```go
var cmd = &plugin.Plugin{
    Command:     []string{"hola"},
    Description: "Saluda al usuario",
    Category:    "general",
    Run: func(client *whatsmeow.Client, m *simple.Message, ctx plugin.CommandContext) error {
        return m.Reply("¡Hola!")
    },
}
```
