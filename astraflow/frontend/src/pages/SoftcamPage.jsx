import React, { useCallback, useEffect, useState } from "react";
import {
    Box, Chip, CircularProgress, Container, FormControl, IconButton,
    InputLabel, MenuItem, Select, Stack, Table, TableBody, TableCell,
    TableHead, TableRow, TextField, Tooltip, Typography
} from "@mui/material";
import SaveIcon from "@mui/icons-material/Save";
import VpnKeyIcon from "@mui/icons-material/VpnKey";
import AuthGuard from "components/Auth/AuthGuard";
import { sendDataToServer } from "utils/functions";

const BISS_MODES = [
    { value: 0, label: "None" },
    { value: 1, label: "BISS-1" },
    { value: 2, label: "BISS-E" },
];

const KEY_LEN = { 0: 0, 1: 16, 2: 32 };

function StreamRow({ stream, onSaved }) {
    const [mode, setMode] = useState(stream.biss_mode || 0);
    const [key, setKey]   = useState(stream.biss_key || "");
    const [saving, setSaving] = useState(false);
    const [dirty, setDirty] = useState(false);

    const expectedLen = KEY_LEN[mode];
    const keyValid = mode === 0 || (key.length === expectedLen && /^[0-9A-Fa-f]+$/.test(key));

    const handleSave = () => {
        setSaving(true);
        sendDataToServer({
            op: "saveSoftcam",
            stream_id: String(stream.id),
            biss_mode: String(mode),
            biss_key: key,
        }).then(res => {
            setSaving(false);
            setDirty(false);
            if (res.status !== "OK") alert(res.status);
            else onSaved();
        });
    };

    return (
        <TableRow hover>
            <TableCell sx={{ fontSize: 13 }}>{stream.name}</TableCell>
            <TableCell>
                <Chip size="small" label={stream.type || "spts"}
                    variant="outlined" sx={{ fontSize: 11 }} />
            </TableCell>
            <TableCell>
                <FormControl size="small" sx={{ minWidth: 100 }}>
                    <Select value={mode} onChange={e => { setMode(e.target.value); setKey(""); setDirty(true); }}>
                        {BISS_MODES.map(m => (
                            <MenuItem key={m.value} value={m.value}>{m.label}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </TableCell>
            <TableCell>
                {mode > 0 && (
                    <TextField
                        size="small"
                        value={key}
                        placeholder={mode === 1 ? "16 hex chars" : "32 hex chars"}
                        inputProps={{ maxLength: expectedLen, style: { fontFamily: "monospace", fontSize: 13 } }}
                        error={key.length > 0 && !keyValid}
                        sx={{ width: mode === 1 ? 170 : 300 }}
                        onChange={e => { setKey(e.target.value.toUpperCase()); setDirty(true); }}
                    />
                )}
            </TableCell>
            <TableCell align="right">
                <Tooltip title="Save">
                    <span>
                        <IconButton size="small" color="primary"
                            disabled={!dirty || saving || (mode > 0 && !keyValid)}
                            onClick={handleSave}>
                            <SaveIcon fontSize="small" />
                        </IconButton>
                    </span>
                </Tooltip>
            </TableCell>
        </TableRow>
    );
}

function NodeBlock({ nodeData, onSaved }) {
    const { node, streams } = nodeData;
    const online = node.status === "online";
    if (!streams || streams.length === 0) return null;

    return (
        <Box mb={4}>
            <Stack direction="row" spacing={1.5} alignItems="center" mb={1.5}>
                <Chip size="small" label={online ? "online" : "offline"}
                    color={online ? "success" : "default"}
                    sx={{ fontWeight: 700, fontSize: 10 }} />
                <Typography variant="subtitle1" fontWeight={800}>
                    {node.name || node.address}
                </Typography>
                <Chip size="small" label={`${streams.length} streams`} variant="outlined" sx={{ fontSize: 11 }} />
            </Stack>
            <Table size="small">
                <TableHead>
                    <TableRow>
                        <TableCell>Stream</TableCell>
                        <TableCell>Type</TableCell>
                        <TableCell>BISS Mode</TableCell>
                        <TableCell>Key</TableCell>
                        <TableCell align="right" />
                    </TableRow>
                </TableHead>
                <TableBody>
                    {streams.map(s => (
                        <StreamRow key={s.id} stream={s} onSaved={onSaved} />
                    ))}
                </TableBody>
            </Table>
        </Box>
    );
}

function SoftcamContent() {
    const [data, setData] = useState(null);

    const load = useCallback(() => {
        sendDataToServer({ op: "getSoftcam" }).then(res => {
            if (res.status === "OK") setData(res.nodes || []);
        });
    }, []);

    useEffect(() => { load(); }, [load]);

    if (data === null) {
        return <Box display="flex" justifyContent="center" pt={8}><CircularProgress /></Box>;
    }

    const hasStreams = data.some(nd => nd.streams?.length > 0);

    return (
        <Container maxWidth="xl" sx={{ py: 3 }}>
            <Stack direction="row" spacing={2} alignItems="center" mb={3}>
                <VpnKeyIcon />
                <Typography variant="h6" fontWeight={700}>Softcam / CA</Typography>
                <Typography variant="body2" sx={{ color: "text.disabled" }}>
                    Configure BISS keys per stream
                </Typography>
            </Stack>
            {!hasStreams ? (
                <Typography color="text.disabled">No streams configured</Typography>
            ) : (
                data.map(nd => (
                    <NodeBlock key={nd.node.id} nodeData={nd} onSaved={load} />
                ))
            )}
        </Container>
    );
}

export default function SoftcamPage() {
    return <AuthGuard><SoftcamContent /></AuthGuard>;
}
