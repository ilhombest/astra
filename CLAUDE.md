# CLAUDE.md — Astra API Bridge

## Контекст одной строкой
Go HTTP сервер эмулирующий API коммерческой Cesbo Astra для связки open source Astra 4.199 (C/Lua) с AstraFlow (Go+React UI).

## Структура репо
```
/
├── ASTRA/          # open source Astra 4.199 (бинарник: /usr/bin/astra)
├── astraflow/      # UI на Go+React, слушает :9000
└── astra-api/      # [СОЗДАТЬ] API Bridge, слушает :8000
```

## Задача
Создать `astra-api/` — Go модуль реализующий HTTP API который:
- AstraFlow вызывает через `POST /control/` и `GET /api/`
- Управляет процессом `astra --stream` через exec/kill/pidfile
- Конвертирует JSON конфиг ↔ Lua скрипт

---

## API контракт (что ожидает AstraFlow)

### POST /control/
```
{"cmd":"version"}               → {"version":"4.4.199","commit":"opensource"}
{"cmd":"load"}                  → {полный JSON конфиг}
{"cmd":"upload","config":{...}} → {"status":true}
{"cmd":"restart"}               → {"status":true}
{"cmd":"sessions"}              → {"sessions":[...]}
```

### GET /api/{path}
```
/api/system-status           → {"cpu":12.5,"mem":45.2,"uptime":3600}
/api/stream-status/{id}?t=0  → {"id":"...","input":{...},"output":[...]}
```

Auth: HTTP Basic Auth.

## JSON конфиг формат
```json
{
  "streams": {
    "id1": {
      "id":"id1","name":"Ch1","enable":true,"type":"spts",
      "input":["udp://239.0.0.1:1234"],
      "output":["http://0.0.0.0:8001/play"]
    }
  },
  "settings": {"http":{"port":8000,"login":"admin","password":"admin"}}
}
```

## Lua конфиг формат (генерировать из JSON)
```lua
-- /etc/astra/astra.lua
make_stream({
  name="Ch1", id="id1", enable=true,
  input={"udp://239.0.0.1:1234"},
  output={"http://0.0.0.0:8001/play"}
})
```

## Файловая структура astra-api/
```
astra-api/
├── main.go      # флаги --port --login --password --config --astra-bin
├── control.go   # POST /control/ → version/load/upload/restart/sessions
├── api.go       # GET /api/system-status, /api/stream-status/{id}
├── process.go   # start/stop/restart astra процесса, pidfile
├── config.go    # JSON↔Lua конвертация, чтение/запись файла
├── auth.go      # Basic Auth middleware
└── go.mod       # module astra-api, go 1.24
```

## Окружение сервера
- OS: Debian Linux
- Astra бинарник: `/usr/bin/astra`
- Lua конфиг: `/etc/astra/astra.lua`
- PID файл: `/var/run/astra.pid`
- AstraFlow: `/usr/src/astraflow/astraflow` → порт 9000

---

## SKILLS

### SKILL: add-stream
**Когда:** добавить новый стрим в конфиг
**Минимальный промт:**
```
add stream: name="{name}" input="{url}" output="{url}" [type=spts|mpts]
```
Что делает: добавляет запись в JSON конфиг, генерирует UUID id, конвертирует в Lua, перезапускает astra через SIGHUP.

### SKILL: config-convert
**Когда:** нужна конвертация между форматами
**Минимальный промт:**
```
convert: lua→json | json→lua
```
Правила:
- `make_stream({...})` → запись в `streams[id]`
- Lua массивы `{...}` → JSON arrays
- отсутствующий id → генерировать UUID

### SKILL: debug-connection
**Когда:** AstraFlow не видит ноду
**Чеклист:**
```bash
# 1. astra-api запущен?
curl -u admin:admin http://localhost:8000/control/ -d '{"cmd":"version"}'
# 2. astra процесс жив?
cat /var/run/astra.pid | xargs ps -p
# 3. конфиг валиден?
cat /etc/astra/astra.lua
# 4. логи
journalctl -u astra-api -n 50
```

### SKILL: systemd-setup
```ini
# /etc/systemd/system/astra-api.service
[Unit]
Description=Astra API Bridge
After=network.target

[Service]
ExecStart=/usr/local/bin/astra-api --port 8000 --login admin --password admin
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```
```bash
systemctl daemon-reload && systemctl enable --now astra-api
```

---

## Правила для Claude Code (экономия токенов)

- Не читай весь файл если нужна одна функция — используй grep
- Не объясняй что делаешь — просто делай
- Ошибки компиляции — исправляй без вопросов
- Тесты — только curl команды, не писать Go тесты
- Комментарии в коде — только на конвертации Lua↔JSON

## Минимальные промты для типичных задач

| Задача | Промт |
|--------|-------|
| Добавить стрим | `add stream name=X input=udp://... output=http://...` |
| Список стримов | `list streams` |
| Рестарт Astra | `restart astra` |
| Проверить связку | `test astraflow→astra-api` |
| Новый эндпоинт | `add endpoint GET /api/X → returns {...}` |
| Починить ошибку | `fix: {текст ошибки}` |

## Приоритет реализации
1. `main.go` + `auth.go` + HTTP сервер
2. `control.go` → version, load, upload, restart
3. `config.go` → JSON↔Lua
4. `process.go` → управление процессом astra
5. `api.go` → system-status, stream-status
6. Systemd сервис
