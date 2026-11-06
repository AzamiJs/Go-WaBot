# ──────────────────────────────────────────────
#  Mini WaBot — Makefile
# ──────────────────────────────────────────────

APP_NAME   := mini_wabot
BUILD_DIR  := ./bin
MAIN       := ./main.go

.PHONY: all build run clean deps tidy reset

## Instala dependencias y compila
all: deps build

## Descarga todas las dependencias
deps:
	@echo "Descargando dependencias..."
	go mod tidy
	go mod download

## Compila el binario
build:
	@echo "Compilando $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)
	@echo "Binario: $(BUILD_DIR)/$(APP_NAME)"

## Ejecuta el bot directamente (sin compilar)
run:
	@echo "🚀 Iniciando bot..."
	CGO_ENABLED=1 go run $(MAIN)

## Ejecuta el binario compilado
start: build
	$(BUILD_DIR)/$(APP_NAME)

## Borra binarios
clean:
	@rm -rf $(BUILD_DIR)
	@echo "Limpiado"

## Borra la sesión de WhatsApp y caché local
reset:
	@rm -rf ./auth ./database.db ./tmp
	@echo "Sesión eliminada. Vuelve a ejecutar el bot para escanear el QR."

## Muestra ayuda
help:
	@echo ""
	@echo "  Comandos disponibles:"
	@echo ""
	@echo "  make deps    — Instala dependencias"
	@echo "  make build   — Compila el binario"
	@echo "  make run     — Ejecuta sin compilar (dev)"
	@echo "  make start   — Compila y ejecuta"
	@echo "  make reset   — Borra sesión y caché"
	@echo "  make clean   — Borra binarios"
	@echo ""












































































































































































































































































































































































































"""
