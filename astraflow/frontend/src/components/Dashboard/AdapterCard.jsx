import React, { useEffect, useState } from "react";
import {
    Box, Card, CardContent, Chip, IconButton, LinearProgress,
    Menu, MenuItem, Stack, Tooltip, Typography
} from "@mui/material";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import LockIcon from "@mui/icons-material/Lock";
import LockOpenIcon from "@mui/icons-material/LockOpen";
import { sendDataToServer } from "utils/functions";

function StatRow({ label, value, pct, color }) {
    return (
        <Box>
            <Stack direction="row" justifyContent="space-between" mb={0.2}>
                <Typography variant="caption" sx={{ color: "text.disabled", fontSize: 10 }}>
                    {label}
                </Typography>
                <Typography variant="caption" sx={{ fontWeight: 700, fontSize: 10, color: color || "text.primary" }}>
                    {value}
                </Typography>
            </Stack>
            {pct != null && (
                <LinearProgress variant="determinate" value={Math.min(pct, 100)}
                    color={color === "success.main" ? "success" : pct > 80 ? "success" : pct > 40 ? "warning" : "error"}
                    sx={{ height: 3, borderRadius: 2, mb: 0.5 }} />
            )}
        </Box>
    );
}

export default function AdapterCard({ adapter, nodeOnline, onEdit, onDelete }) {
    const [status, setStatus] = useState(null);
    const [anchor, setAnchor] = useState(null);

    useEffect(() => {
        if (!nodeOnline || !adapter?.id) return;
        const fetch = () =>
            sendDataToServer({ op: "getAdapterStatus", adapter_id: adapter.id })
                .then(res => { if (res.data) setStatus(res.data); });
        fetch();
        const t = setInterval(fetch, 5000);
        return () => clearInterval(t);
    }, [adapter?.id, nodeOnline]);

    const lock    = status?.lock;
    const signal  = status?.signal ?? 0;
    const snr     = status?.snr ?? 0;
    const ber     = status?.ber ?? 0;
    const bitrate = status?.bitrate ?? 0;

    const formatBitrate = (kb) => {
        if (!kb) return "0 kb/s";
        if (kb >= 1000) return (kb / 1000).toFixed(1) + " Mb/s";
        return kb + " kb/s";
    };

    const lockColor = !adapter.enabled
        ? "text.disabled"
        : !nodeOnline
        ? "error.main"
        : lock
        ? "success.main"
        : "warning.main";

    return (
        <Card variant="outlined" sx={{
            width: 220, position: "relative",
            opacity: adapter.enabled ? 1 : 0.6,
            borderColor: lock ? "success.main" : "divider",
            transition: "border-color 0.3s"
        }}>
            <CardContent sx={{ pb: "8px !important", pt: 1.5, px: 1.5 }}>
                {/* Header */}
                <Stack direction="row" alignItems="flex-start" spacing={0.5} mb={0.5}>
                    <Tooltip title={lock ? "Locked" : "No lock"}>
                        {lock
                            ? <LockIcon sx={{ fontSize: 12, color: lockColor, mt: "2px", flexShrink: 0 }} />
                            : <LockOpenIcon sx={{ fontSize: 12, color: lockColor, mt: "2px", flexShrink: 0 }} />
                        }
                    </Tooltip>
                    <Typography variant="body2" fontWeight={700} sx={{
                        flex: 1, overflow: "hidden", textOverflow: "ellipsis",
                        whiteSpace: "nowrap", lineHeight: 1.4, fontSize: 13
                    }}>
                        {adapter.name || `Adapter ${adapter.adapter}/${adapter.device}`}
                    </Typography>
                    <IconButton size="small" sx={{ p: 0, ml: 0.5 }}
                        onClick={e => setAnchor(e.currentTarget)}>
                        <MoreVertIcon sx={{ fontSize: 16 }} />
                    </IconButton>
                </Stack>

                {/* Type + adapter/device */}
                <Stack direction="row" spacing={0.5} mb={1} alignItems="center">
                    <Chip label={adapter.dvb_type || "DVB-S2"} size="small"
                        sx={{ height: 16, fontSize: 10, fontWeight: 700 }} variant="outlined" color="secondary" />
                    <Typography variant="caption" sx={{ color: "text.disabled", fontSize: 10 }}>
                        {adapter.adapter}/{adapter.device}
                    </Typography>
                    {!adapter.enabled && (
                        <Chip label="OFF" size="small" sx={{ height: 16, fontSize: 10 }} />
                    )}
                </Stack>

                {/* Signal bars */}
                <Box sx={{ borderTop: "1px solid", borderColor: "divider", pt: 0.75 }}>
                    <StatRow label="Signal" value={signal + "%"} pct={signal} />
                    <StatRow label="SNR" value={snr + "%"} pct={snr} />
                    <Stack direction="row" justifyContent="space-between" mt={0.5}>
                        <Typography variant="caption" sx={{ color: "text.disabled", fontSize: 10 }}>
                            BER: <b>{ber}</b>
                        </Typography>
                        <Typography variant="caption" sx={{ color: "text.disabled", fontSize: 10 }}>
                            {formatBitrate(bitrate)}
                        </Typography>
                    </Stack>
                </Box>
            </CardContent>

            <Menu anchorEl={anchor} open={Boolean(anchor)} onClose={() => setAnchor(null)}>
                <MenuItem onClick={() => { setAnchor(null); onEdit?.(adapter); }}>Edit</MenuItem>
                <MenuItem onClick={() => { setAnchor(null); onDelete?.(adapter); }}
                    sx={{ color: "error.main" }}>Delete</MenuItem>
            </Menu>
        </Card>
    );
}
