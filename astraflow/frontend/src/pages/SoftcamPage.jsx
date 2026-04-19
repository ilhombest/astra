import React, { useCallback, useEffect, useState } from "react";
import {
    Box, Button, Chip, CircularProgress, Container, Dialog, DialogActions,
    DialogContent, DialogTitle, FormControl, FormControlLabel, IconButton,
    InputLabel, MenuItem, Select, Stack, Switch, Tab, Tabs, Table, TableBody,
    TableCell, TableHead, TableRow, TextField, Tooltip, Typography
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import SaveIcon from "@mui/icons-material/Save";
import VpnKeyIcon from "@mui/icons-material/VpnKey";
import RouterIcon from "@mui/icons-material/Router";
import AuthGuard from "components/Auth/AuthGuard";
import { sendDataToServer } from "utils/functions";

// ── BISS tab ──────────────────────────────────────────────────────────────────

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
    const [dirty, setDirty]   = useState(false);

    const expectedLen = KEY_LEN[mode];
    const keyValid = mode === 0 || (key.length === expectedLen && /^[0-9A-Fa-f]+$/.test(key));

    const handleSave = () => {
        setSaving(true);
        sendDataToServer({ op: "saveSoftcam", stream_id: String(stream.id),
            biss_mode: String(mode), biss_key: key }).then(res => {
            setSaving(false); setDirty(false);
            if (res.status !== "OK") alert(res.status); else onSaved();
        });
    };

    return (
        <TableRow hover>
            <TableCell sx={{ fontSize: 13 }}>{stream.name}</TableCell>
            <TableCell>
                <Chip size="small" label={stream.type || "spts"} variant="outlined" sx={{ fontSize: 11 }} />
            </TableCell>
            <TableCell>
                <FormControl size="small" sx={{ minWidth: 100 }}>
                    <Select value={mode} onChange={e => { setMode(e.target.value); setKey(""); setDirty(true); }}>
                        {BISS_MODES.map(m => <MenuItem key={m.value} value={m.value}>{m.label}</MenuItem>)}
                    </Select>
                </FormControl>
            </TableCell>
            <TableCell>
                {mode > 0 && (
                    <TextField size="small" value={key}
                        placeholder={mode === 1 ? "16 hex chars" : "32 hex chars"}
                        inputProps={{ maxLength: expectedLen, style: { fontFamily: "monospace", fontSize: 13 } }}
                        error={key.length > 0 && !keyValid}
                        sx={{ width: mode === 1 ? 170 : 300 }}
                        onChange={e => { setKey(e.target.value.toUpperCase()); setDirty(true); }} />
                )}
            </TableCell>
            <TableCell align="right">
                <Tooltip title="Save"><span>
                    <IconButton size="small" color="primary"
                        disabled={!dirty || saving || (mode > 0 && !keyValid)}
                        onClick={handleSave}>
                        <SaveIcon fontSize="small" />
                    </IconButton>
                </span></Tooltip>
            </TableCell>
        </TableRow>
    );
}

function BissTab({ data, onRefresh }) {
    const hasStreams = data.some(nd => nd.streams?.length > 0);
    if (!hasStreams) return <Typography color="text.disabled" mt={2}>No streams configured</Typography>;
    return (
        <>
            {data.map(nd => {
                const { node, streams } = nd;
                if (!streams?.length) return null;
                return (
                    <Box key={nd.node.id} mb={4}>
                        <Stack direction="row" spacing={1.5} alignItems="center" mb={1.5}>
                            <Chip size="small" label={node.status === "online" ? "online" : "offline"}
                                color={node.status === "online" ? "success" : "default"}
                                sx={{ fontWeight: 700, fontSize: 10 }} />
                            <Typography variant="subtitle1" fontWeight={800}>{node.name || node.address}</Typography>
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
                                {streams.map(s => <StreamRow key={s.id} stream={s} onSaved={onRefresh} />)}
                            </TableBody>
                        </Table>
                    </Box>
                );
            })}
        </>
    );
}

// ── NewCamd tab ───────────────────────────────────────────────────────────────

const NEWCAMD_EMPTY = { id: 0, node_id: "", name: "", host: "", port: 2222,
    username: "", password: "", des_key: "", enabled: true };

function NewcamdDialog({ open, initial, nodeId, onClose, onSaved }) {
    const [form, setForm] = useState(NEWCAMD_EMPTY);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        if (open) setForm(initial ? { ...initial } : { ...NEWCAMD_EMPTY, node_id: nodeId });
    }, [open, initial, nodeId]);

    const set = (field) => (e) => setForm(f => ({ ...f, [field]: e.target.value }));

    const desValid = form.des_key === "" || (form.des_key.length === 28 && /^[0-9A-Fa-f]+$/.test(form.des_key));

    const handleSave = () => {
        if (!form.host || !form.port) return;
        setSaving(true);
        sendDataToServer({
            op: "saveNewcamd",
            id: String(form.id || 0),
            node_id: form.node_id || nodeId,
            name: form.name,
            host: form.host,
            port: String(form.port),
            username: form.username,
            password: form.password,
            des_key: form.des_key,
            enabled: form.enabled ? "1" : "0",
        }).then(res => {
            setSaving(false);
            if (res.status !== "OK") alert(res.status);
            else { onSaved(); onClose(); }
        });
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{form.id ? "Edit NewCamd Server" : "Add NewCamd Server"}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} mt={1}>
                    <TextField label="Name" size="small" fullWidth value={form.name} onChange={set("name")}
                        placeholder="e.g. OSCam main" />
                    <Stack direction="row" spacing={2}>
                        <TextField label="Host" size="small" fullWidth value={form.host} onChange={set("host")}
                            required placeholder="192.168.1.100" />
                        <TextField label="Port" size="small" sx={{ width: 110 }} type="number"
                            value={form.port} onChange={set("port")} required inputProps={{ min: 1, max: 65535 }} />
                    </Stack>
                    <Stack direction="row" spacing={2}>
                        <TextField label="Username" size="small" fullWidth value={form.username} onChange={set("username")} />
                        <TextField label="Password" size="small" fullWidth type="password"
                            value={form.password} onChange={set("password")} />
                    </Stack>
                    <TextField
                        label="DES Key (14 bytes / 28 hex chars)"
                        size="small" fullWidth
                        value={form.des_key}
                        placeholder="0102030405060708091011121314"
                        inputProps={{ maxLength: 28, style: { fontFamily: "monospace" } }}
                        error={form.des_key.length > 0 && !desValid}
                        helperText={form.des_key.length > 0 && !desValid ? "Must be exactly 28 hex characters" : ""}
                        onChange={e => setForm(f => ({ ...f, des_key: e.target.value.toUpperCase() }))}
                    />
                    <FormControlLabel
                        control={<Switch checked={form.enabled}
                            onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))} />}
                        label="Enabled" />
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
                <Button variant="contained" onClick={handleSave}
                    disabled={saving || !form.host || !form.port || (form.des_key && !desValid)}>
                    Save
                </Button>
            </DialogActions>
        </Dialog>
    );
}

function NewcamdTab({ data, onRefresh }) {
    const [dialogOpen, setDialogOpen] = useState(false);
    const [editServer, setEditServer] = useState(null);
    const [editNodeId, setEditNodeId] = useState("");

    const handleAdd = (nodeId) => { setEditServer(null); setEditNodeId(nodeId); setDialogOpen(true); };
    const handleEdit = (srv) => { setEditServer(srv); setEditNodeId(srv.node_id); setDialogOpen(true); };
    const handleDelete = (srv) => {
        if (!window.confirm(`Delete NewCamd server "${srv.name || srv.host}"?`)) return;
        sendDataToServer({ op: "deleteNewcamd", id: String(srv.id) }).then(res => {
            if (res.status !== "OK") alert(res.status); else onRefresh();
        });
    };

    return (
        <>
            {data.map(nd => {
                const { node, servers } = nd;
                return (
                    <Box key={nd.node.id} mb={4}>
                        <Stack direction="row" spacing={1.5} alignItems="center" mb={1.5}>
                            <Chip size="small" label={node.status === "online" ? "online" : "offline"}
                                color={node.status === "online" ? "success" : "default"}
                                sx={{ fontWeight: 700, fontSize: 10 }} />
                            <Typography variant="subtitle1" fontWeight={800}>{node.name || node.address}</Typography>
                            <Box flex={1} />
                            <Button size="small" startIcon={<AddIcon />}
                                onClick={() => handleAdd(node.node_id)}>
                                Add server
                            </Button>
                        </Stack>
                        {!servers?.length ? (
                            <Typography variant="body2" color="text.disabled" pl={1}>
                                No NewCamd servers configured
                            </Typography>
                        ) : (
                            <Table size="small">
                                <TableHead>
                                    <TableRow>
                                        <TableCell>Name</TableCell>
                                        <TableCell>Host : Port</TableCell>
                                        <TableCell>Username</TableCell>
                                        <TableCell>DES Key</TableCell>
                                        <TableCell>Status</TableCell>
                                        <TableCell align="right" />
                                    </TableRow>
                                </TableHead>
                                <TableBody>
                                    {servers.map(srv => (
                                        <TableRow key={srv.id} hover>
                                            <TableCell sx={{ fontSize: 13 }}>{srv.name || "—"}</TableCell>
                                            <TableCell sx={{ fontFamily: "monospace", fontSize: 12 }}>
                                                {srv.host}:{srv.port}
                                            </TableCell>
                                            <TableCell sx={{ fontSize: 12 }}>{srv.username || "—"}</TableCell>
                                            <TableCell sx={{ fontFamily: "monospace", fontSize: 11, color: "text.disabled" }}>
                                                {srv.des_key ? `${srv.des_key.slice(0, 8)}…` : "—"}
                                            </TableCell>
                                            <TableCell>
                                                <Chip size="small"
                                                    label={srv.enabled ? "enabled" : "disabled"}
                                                    color={srv.enabled ? "success" : "default"}
                                                    sx={{ fontSize: 10 }} />
                                            </TableCell>
                                            <TableCell align="right">
                                                <IconButton size="small" onClick={() => handleEdit(srv)}>
                                                    <EditIcon fontSize="small" />
                                                </IconButton>
                                                <IconButton size="small" color="error" onClick={() => handleDelete(srv)}>
                                                    <DeleteIcon fontSize="small" />
                                                </IconButton>
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        )}
                    </Box>
                );
            })}
            <NewcamdDialog open={dialogOpen} initial={editServer} nodeId={editNodeId}
                onClose={() => setDialogOpen(false)} onSaved={onRefresh} />
        </>
    );
}

// ── Page ──────────────────────────────────────────────────────────────────────

function SoftcamContent() {
    const [tab, setTab]         = useState(0);
    const [bissData, setBiss]   = useState(null);
    const [ncData, setNC]       = useState(null);

    const loadBiss = useCallback(() => {
        sendDataToServer({ op: "getSoftcam" }).then(res => {
            if (res.status === "OK") setBiss(res.nodes || []);
        });
    }, []);

    const loadNC = useCallback(() => {
        sendDataToServer({ op: "getNewcamd" }).then(res => {
            if (res.status === "OK") setNC(res.nodes || []);
        });
    }, []);

    useEffect(() => { loadBiss(); loadNC(); }, [loadBiss, loadNC]);

    const loading = bissData === null || ncData === null;

    return (
        <Container maxWidth="xl" sx={{ py: 3 }}>
            <Stack direction="row" spacing={2} alignItems="center" mb={2}>
                <VpnKeyIcon />
                <Typography variant="h6" fontWeight={700}>Softcam / CA</Typography>
            </Stack>

            <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 3, borderBottom: 1, borderColor: "divider" }}>
                <Tab icon={<VpnKeyIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="BISS Keys" />
                <Tab icon={<RouterIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="NewCamd" />
            </Tabs>

            {loading ? (
                <Box display="flex" justifyContent="center" pt={6}><CircularProgress /></Box>
            ) : tab === 0 ? (
                <BissTab data={bissData} onRefresh={loadBiss} />
            ) : (
                <NewcamdTab data={ncData} onRefresh={loadNC} />
            )}
        </Container>
    );
}

export default function SoftcamPage() {
    return <AuthGuard><SoftcamContent /></AuthGuard>;
}
