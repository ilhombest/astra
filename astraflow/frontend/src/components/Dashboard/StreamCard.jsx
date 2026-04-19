import React, { useEffect, useState } from "react";
import {
    Box, Card, CardContent, Chip, IconButton, Menu, MenuItem,
    Stack, Tooltip, Typography
} from "@mui/material";
import MoreVertIcon from "@mui/icons-material/MoreVert";
import FiberManualRecordIcon from "@mui/icons-material/FiberManualRecord";
import { sendDataToServer } from "utils/functions";

function Metric({ label, value, color }) {
    return (
        <Box sx={{ textAlign: "center", minWidth: 40 }}>
            <Typography variant="caption" sx={{ color: "text.disabled", display: "block", lineHeight: 1.2 }}>
                {label}
            </Typography>
            <Typography variant="caption" sx={{ fontWeight: 700, color: color || "text.primary" }}>
                {value ?? "—"}
            </Typography>
        </Box>
    );
}

export default function StreamCard({ stream, nodeOnline, onEdit, onDelete }) {
    const [status, setStatus] = useState(null);
    const [anchor, setAnchor] = useState(null);

    useEffect(() => {
        if (!nodeOnline || !stream?.id) return;
        const fetch = () =>
            sendDataToServer({ op: "getStreamStatus", stream_id: stream.id })
                .then(res => { if (res.data) setStatus(res.data); });
        fetch();
        const t = setInterval(fetch, 5000);
        return () => clearInterval(t);
    }, [stream?.id, nodeOnline]);

    const active = status?.active;
    const onair  = status?.onair;
    const bitrate = status?.bitrate;
    const sessions = status?.sessions;
    const ccErr  = status?.cc_error;
    const pesErr = status?.pes_error;
    const scErr  = status?.sc_error;

    const dotColor = !stream.enable
        ? "text.disabled"
        : !nodeOnline
        ? "error.main"
        : onair
        ? "success.main"
        : active
        ? "warning.main"
        : "text.disabled";

    const formatBitrate = (kb) => {
        if (kb == null) return "—";
        if (kb >= 1000) return (kb / 1000).toFixed(1) + " Mb/s";
        return kb + " kb/s";
    };

    return (
        <Card variant="outlined" sx={{
            width: 220, position: "relative",
            opacity: stream.enable ? 1 : 0.6,
            borderColor: onair ? "success.main" : "divider",
            transition: "border-color 0.3s"
        }}>
            <CardContent sx={{ pb: "8px !important", pt: 1.5, px: 1.5 }}>
                {/* Header */}
                <Stack direction="row" alignItems="flex-start" spacing={0.5} mb={0.5}>
                    <Tooltip title={onair ? "On Air" : active ? "Active" : "Inactive"}>
                        <FiberManualRecordIcon sx={{ fontSize: 10, color: dotColor, mt: "3px", flexShrink: 0 }} />
                    </Tooltip>
                    <Typography variant="body2" fontWeight={700} sx={{
                        flex: 1, overflow: "hidden", textOverflow: "ellipsis",
                        whiteSpace: "nowrap", lineHeight: 1.4, fontSize: 13
                    }}>
                        {stream.name || "Untitled"}
                    </Typography>
                    <IconButton size="small" sx={{ p: 0, ml: 0.5 }}
                        onClick={e => setAnchor(e.currentTarget)}>
                        <MoreVertIcon sx={{ fontSize: 16 }} />
                    </IconButton>
                </Stack>

                {/* Type chip */}
                <Stack direction="row" spacing={0.5} mb={1}>
                    <Chip label={stream.type?.toUpperCase() || "SPTS"} size="small"
                        sx={{ height: 16, fontSize: 10, fontWeight: 700 }} variant="outlined" />
                    {!stream.enable && (
                        <Chip label="OFF" size="small" color="default"
                            sx={{ height: 16, fontSize: 10 }} />
                    )}
                </Stack>

                {/* Metrics row */}
                <Stack direction="row" spacing={0.5} justifyContent="space-between"
                    sx={{ borderTop: "1px solid", borderColor: "divider", pt: 0.75 }}>
                    <Metric label="Bitrate" value={formatBitrate(bitrate)} />
                    <Metric label="Sessions" value={sessions ?? "—"} />
                    <Metric label="PES" value={pesErr ?? "—"}
                        color={pesErr > 0 ? "error.main" : undefined} />
                    <Metric label="CC" value={ccErr ?? "—"}
                        color={ccErr > 0 ? "error.main" : undefined} />
                    {scErr > 0 && <Metric label="SC" value={scErr} color="warning.main" />}
                </Stack>
            </CardContent>

            <Menu anchorEl={anchor} open={Boolean(anchor)} onClose={() => setAnchor(null)}>
                <MenuItem onClick={() => { setAnchor(null); onEdit?.(stream); }}>Edit</MenuItem>
                <MenuItem onClick={() => { setAnchor(null); onDelete?.(stream); }}
                    sx={{ color: "error.main" }}>Delete</MenuItem>
            </Menu>
        </Card>
    );
}
