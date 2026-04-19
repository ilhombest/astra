import React, { useCallback, useEffect, useRef, useState } from "react";
import {
    Box, Chip, CircularProgress, Container, FormControl,
    FormControlLabel, InputLabel, MenuItem, Select, Stack,
    Switch, Typography
} from "@mui/material";
import ArticleIcon from "@mui/icons-material/Article";
import AuthGuard from "components/Auth/AuthGuard";
import { sendDataToServer } from "utils/functions";

const LEVEL_COLOR = {
    error:   "#f44336",
    warning: "#ff9800",
    info:    "#9e9e9e",
};

function LogLine({ entry }) {
    const color = LEVEL_COLOR[entry.level] || LEVEL_COLOR.info;
    return (
        <Box component="div" sx={{ display: "flex", gap: 1.5, lineHeight: 1.6 }}>
            <Box component="span" sx={{ color: "text.disabled", flexShrink: 0, fontSize: 11 }}>
                {entry.time}
            </Box>
            <Box component="span" sx={{ color, flexShrink: 0, fontSize: 11, minWidth: 52 }}>
                [{entry.level}]
            </Box>
            <Box component="span" sx={{ fontSize: 12, wordBreak: "break-all" }}>
                {entry.msg}
            </Box>
        </Box>
    );
}

function LogContent() {
    const [nodes, setNodes]       = useState(null);
    const [nodeId, setNodeId]     = useState("");
    const [lines, setLines]       = useState([]);
    const [autoScroll, setAutoScroll] = useState(true);
    const bottomRef = useRef(null);

    useEffect(() => {
        sendDataToServer({ op: "getDashboardData" }).then(res => {
            const ns = (res.nodes || []).map(nd => nd.node);
            setNodes(ns);
            if (ns.length > 0) setNodeId(ns[0].node_id);
        });
    }, []);

    const load = useCallback(() => {
        if (!nodeId) return;
        sendDataToServer({ op: "getNodeLog", node_id: nodeId }).then(res => {
            if (Array.isArray(res.lines)) setLines(res.lines);
        });
    }, [nodeId]);

    useEffect(() => {
        load();
        const t = setInterval(load, 3000);
        return () => clearInterval(t);
    }, [load]);

    useEffect(() => {
        if (autoScroll && bottomRef.current) {
            bottomRef.current.scrollIntoView({ behavior: "smooth" });
        }
    }, [lines, autoScroll]);

    if (nodes === null) {
        return <Box display="flex" justifyContent="center" pt={8}><CircularProgress /></Box>;
    }

    return (
        <Container maxWidth="xl" sx={{ py: 3 }}>
            <Stack direction="row" spacing={2} alignItems="center" mb={2} flexWrap="wrap">
                <ArticleIcon />
                <Typography variant="h6" fontWeight={700}>Log</Typography>
                <Chip label={`${lines.length} lines`} size="small" />
                {nodes.length > 1 && (
                    <FormControl size="small" sx={{ minWidth: 180 }}>
                        <InputLabel>Node</InputLabel>
                        <Select value={nodeId} label="Node"
                            onChange={e => setNodeId(e.target.value)}>
                            {nodes.map(n => (
                                <MenuItem key={n.node_id} value={n.node_id}>
                                    {n.name || n.address}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                )}
                <FormControlLabel
                    control={<Switch checked={autoScroll} size="small"
                        onChange={e => setAutoScroll(e.target.checked)} />}
                    label={<Typography variant="caption">Auto-scroll</Typography>}
                    sx={{ ml: "auto !important" }}
                />
            </Stack>

            <Box sx={{
                bgcolor: "background.paper",
                border: "1px solid",
                borderColor: "divider",
                borderRadius: 1,
                p: 1.5,
                height: "calc(100vh - 180px)",
                overflowY: "auto",
                fontFamily: "monospace",
            }}>
                {lines.length === 0 ? (
                    <Typography variant="body2" sx={{ color: "text.disabled" }}>
                        No log entries
                    </Typography>
                ) : (
                    lines.map((entry, i) => <LogLine key={i} entry={entry} />)
                )}
                <div ref={bottomRef} />
            </Box>
        </Container>
    );
}

export default function LogPage() {
    return <AuthGuard><LogContent /></AuthGuard>;
}
