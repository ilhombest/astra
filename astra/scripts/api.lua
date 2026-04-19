-- Astra Built-in Web API
-- Replaces the astra-api Go bridge; serves POST /control/ and GET /api/*
-- Compatible with AstraFlow expectations.
--
-- Usage: astra --stream /etc/astra/main.lua
-- main.lua must call: dofile("/path/to/api.lua")

local CONFIG_PATH = os.getenv("ASTRA_CONFIG") or "/etc/astra/astra.json"
local API_LOGIN   = os.getenv("ASTRA_LOGIN")  or "admin"
local API_PASS    = os.getenv("ASTRA_PASS")   or "admin"
local API_PORT    = tonumber(os.getenv("ASTRA_PORT")) or 8000

local active_channels = {}   -- stream_id  → channel_data
local active_cams     = {}   -- cam_id     → cam instance
local start_time      = os.time()

-- -----------------------------------------------------------------------
-- Config application
-- -----------------------------------------------------------------------

local function apply_config(cfg)
    -- Kill live channels first
    for id, ch in pairs(active_channels) do
        kill_channel(ch)
    end
    active_channels = {}

    -- Release CAMs
    for id, _ in pairs(active_cams) do
        active_cams[id] = nil
    end
    active_cams = {}
    collectgarbage()

    -- Create DVB tuners; store as global dvb{N} so dvb://dvb6#pnr=X URLs resolve
    local adapters = cfg.adapters or {}
    for key, a in pairs(adapters) do
        local n        = tonumber(key) or 0
        local dvb_type = tostring(a.dvb_type or "DVB-S2"):gsub("DVB%-", "")

        local conf = {
            type    = dvb_type,
            adapter = tonumber(a.adapter) or n,
        }
        if tonumber(a.device or 0) ~= 0 then
            conf.device = tonumber(a.device)
        end

        if dvb_type == "S" or dvb_type == "S2" then
            conf.tp  = table.concat({ a.frequency or 0, a.polarization or "H", a.symbolrate or 0 }, ":")
            conf.lnb = table.concat({ a.lof1 or 9750, a.lof2 or 10600, a.slof or 11700 }, ":")
        elseif dvb_type == "T" or dvb_type == "T2" or dvb_type == "ATSC" or dvb_type == "ISDB-T" then
            conf.tp = table.concat({ a.frequency or 0, a.bandwidth or 8 }, ":")
        elseif dvb_type == "C" or dvb_type == "C2" then
            conf.tp = table.concat({ a.frequency or 0, a.symbolrate or 0 }, ":")
        end

        _G["dvb" .. n] = dvb_tune(conf)
        log.info("[api] adapter " .. n .. " (" .. dvb_type .. ") tuned")
    end

    -- Create NewCamd CAM instances
    local cams_cfg = cfg.cams or {}
    for cam_id, c in pairs(cams_cfg) do
        if (c.type or "newcamd") == "newcamd" and newcamd then
            active_cams[cam_id] = newcamd({
                name        = c.name or cam_id,
                host        = c.host or "",
                port        = tonumber(c.port) or 2222,
                user        = c.user or "",
                pass        = c.pass or "",
                key         = c.key  or "0102030405060708091011121314",
                disable_emm = c.disable_emm == true,
            })
            log.info("[api] newcamd cam '" .. cam_id .. "' created")
        end
    end

    -- Create channels
    local streams = cfg.streams or {}
    for stream_id, s in pairs(streams) do
        if s.enable ~= false then
            -- Convert dvb://6#pnr=X → dvb://dvb6#pnr=X
            local inputs = {}
            for _, url in ipairs(s.input or {}) do
                url = url:gsub("^dvb://(%d+)", "dvb://dvb%1")
                table.insert(inputs, url)
            end

            local ch_conf = {
                name   = s.name or stream_id,
                id     = stream_id,
                enable = true,
                input  = inputs,
                output = s.output or {},
            }

            if s.cam and active_cams[s.cam] then
                ch_conf.cam = active_cams[s.cam]
            end
            if s.biss and s.biss ~= "" then
                ch_conf.biss = s.biss
            end

            local ch = make_channel(ch_conf)
            if ch then
                active_channels[stream_id] = ch
                log.info("[api] stream '" .. stream_id .. "' started")
            end
        end
    end
end

-- -----------------------------------------------------------------------
-- HTTP helpers
-- -----------------------------------------------------------------------

local function check_auth(request)
    local auth    = (request.headers or {})["authorization"] or ""
    local encoded = auth:match("^[Bb]asic%s+(.+)$")
    if not encoded then return false end
    local decoded = base64.decode(encoded)
    return decoded == API_LOGIN .. ":" .. API_PASS
end

local function send_json(server, client, code, tbl)
    server:send(client, {
        code    = code,
        headers = {
            "Content-Type: application/json",
            "Access-Control-Allow-Origin: *",
            "Access-Control-Allow-Headers: Authorization, Content-Type",
        },
        content = json.encode(tbl),
    })
end

local function unauthorized(server, client)
    server:send(client, {
        code    = 401,
        headers = { 'WWW-Authenticate: Basic realm="Astra"' },
        content = "Unauthorized",
    })
end

-- -----------------------------------------------------------------------
-- POST /control/
-- -----------------------------------------------------------------------

local function on_control(server, client, request)
    if not request then return end

    if not check_auth(request) then
        unauthorized(server, client)
        return
    end

    local ok, req = pcall(json.decode, request.content or "{}")
    if not ok or type(req) ~= "table" then req = {} end
    local cmd = req.cmd or ""

    if cmd == "version" then
        send_json(server, client, 200, { version = "4.4.199", commit = "opensource" })

    elseif cmd == "load" then
        local cfg = json.load(CONFIG_PATH) or { streams = {}, settings = {} }
        send_json(server, client, 200, cfg)

    elseif cmd == "upload" then
        local new_cfg = req.config
        if type(new_cfg) ~= "table" then
            send_json(server, client, 200, { status = false, error = "config missing" })
            return
        end
        json.save(CONFIG_PATH, new_cfg)
        local ok2, err = pcall(apply_config, new_cfg)
        if ok2 then
            send_json(server, client, 200, { status = true })
        else
            log.error("[api] apply_config: " .. tostring(err))
            send_json(server, client, 200, { status = false, error = tostring(err) })
        end

    elseif cmd == "restart" then
        send_json(server, client, 200, { status = true })
        timer({ interval = 1, callback = function(self)
            self:close()
            astra.reload()
        end })

    elseif cmd == "sessions" then
        local sessions = {}
        for id, ch in pairs(active_channels) do
            local on_air = ch.input and ch.input[1] and ch.input[1].on_air == true
            table.insert(sessions, {
                id     = id,
                name   = ch.config and ch.config.name or id,
                on_air = on_air,
            })
        end
        send_json(server, client, 200, { sessions = sessions })

    elseif cmd == "set-stream" then
        -- Dynamic single-stream update (used by some integrations)
        local id     = req.id or ""
        local s      = req.stream
        if type(s) ~= "table" or id == "" then
            send_json(server, client, 200, { status = false, error = "id/stream required" })
            return
        end
        -- Kill existing
        if active_channels[id] then
            kill_channel(active_channels[id])
            active_channels[id] = nil
        end
        -- Delete if stream is null/empty
        if next(s) == nil then
            send_json(server, client, 200, { status = true })
            return
        end
        local inputs = {}
        for _, url in ipairs(s.input or {}) do
            url = url:gsub("^dvb://(%d+)", "dvb://dvb%1")
            table.insert(inputs, url)
        end
        local ch = make_channel({
            name   = s.name or id,
            id     = id,
            enable = s.enable ~= false,
            input  = inputs,
            output = s.output or {},
        })
        if ch then active_channels[id] = ch end
        send_json(server, client, 200, { status = true })

    else
        send_json(server, client, 200, { error = "unknown command: " .. cmd })
    end
end

-- -----------------------------------------------------------------------
-- GET /api/*
-- -----------------------------------------------------------------------

local function on_api(server, client, request)
    if not request then return end

    if not check_auth(request) then
        unauthorized(server, client)
        return
    end

    local path = request.path or ""

    if path == "/api/system-status" then
        send_json(server, client, 200, {
            cpu    = 0,
            mem    = 0,
            uptime = os.time() - start_time,
        })

    else
        -- /api/stream-status/{id}?t=0
        local stream_id = path:match("^/api/stream%-status/(.+)$")
        if stream_id then
            stream_id = stream_id:match("^([^?]+)") or stream_id
        end

        if stream_id and active_channels[stream_id] then
            local ch     = active_channels[stream_id]
            local on_air = ch.input and ch.input[1] and ch.input[1].on_air == true
            send_json(server, client, 200, {
                id     = stream_id,
                input  = { status = on_air and "ok" or "no signal" },
                output = {},
            })
        else
            send_json(server, client, 404, { error = "not found" })
        end
    end
end

-- -----------------------------------------------------------------------
-- Startup: load config, start server
-- -----------------------------------------------------------------------

local init_cfg = json.load(CONFIG_PATH)
if init_cfg then
    local ok, err = pcall(apply_config, init_cfg)
    if not ok then
        log.error("[api] startup config error: " .. tostring(err))
    end
else
    log.warning("[api] no config at " .. CONFIG_PATH .. " — starting empty")
end

http_server({
    addr  = "0.0.0.0",
    port  = API_PORT,
    route = {
        { "/control/", on_control },
        { "/api/*",    on_api    },
    },
})

log.info("[api] web API listening on :" .. API_PORT)
