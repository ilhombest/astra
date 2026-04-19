import React, { useCallback, useEffect, useRef, useState } from "react";
import {
    Box, Button, Chip, CircularProgress, Container, Divider,
    InputAdornment, Stack, TextField, Tooltip, Typography
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import CloseIcon from "@mui/icons-material/Close";
import SearchIcon from "@mui/icons-material/Search";
import SettingsInputAntennaIcon from "@mui/icons-material/SettingsInputAntenna";
import LiveTvIcon from "@mui/icons-material/LiveTv";
import AuthGuard from "components/Auth/AuthGuard";
import StreamCard from "components/Dashboard/StreamCard";
import AdapterCard from "components/Dashboard/AdapterCard";
import { sendDataToServer } from "utils/functions";
import AdaptersDialog from "components/WorkArea/AdaptersDialog";
import InputEditDialog from "components/WorkArea/InputEditDialog";

function NodeSection({ nodeData, onRefresh, searchQuery }) {
    const { node, streams, adapters } = nodeData;
    const online = node.status === "online";

    const [adaptersOpen, setAdaptersOpen] = useState(false);
    const [streamOpen, setStreamOpen] = useState(false);
    const [editStream, setEditStream] = useState(null);

    const q = searchQuery.toLowerCase();
    const filteredStreams  = q ? streams.filter(s => s.name.toLowerCase().includes(q)) : streams;
    const filteredAdapters = q ? adapters.filter(a => a.name.toLowerCase().includes(q)) : adapters;

    // hide node entirely if search active and nothing matches
    if (q && filteredStreams.length === 0 && filteredAdapters.length === 0) return null;

    const handleDeleteStream = (stream) => {
        if (!window.confirm(`Delete stream "${stream.name}"?`)) return;
        sendDataToServer({ op: "deleteStream", id: stream.id }).then(res => {
            if (res.status === "OK") onRefresh();
            else alert(res.status);
        });
    };

    return (
        <Box mb={4}>
            <Stack direction="row" alignItems="center" spacing={1.5} mb={1.5}>
                <Chip size="small" label={online ? "online" : "offline"}
                    color={online ? "success" : "default"}
                    sx={{ fontWeight: 700, fontSize: 10 }} />
                <Typography variant="subtitle1" fontWeight={800}>
                    {node.name || node.address}
                </Typography>
                <Typography variant="caption" sx={{ color: "text.disabled" }}>
                    {node.address}
                </Typography>
                {node.version && (
                    <Typography variant="caption" sx={{ color: "text.disabled" }}>
                        v{node.version}
                    </Typography>
                )}
                <Box flex={1} />
                <Tooltip title="Manage adapters">
                    <Button size="small" startIcon={<SettingsInputAntennaIcon />}
                        onClick={() => setAdaptersOpen(true)}>
                        Adapters
                    </Button>
                </Tooltip>
                <Tooltip title="Add stream">
                    <Button size="small" variant="contained" startIcon={<AddIcon />}
                        onClick={() => {
                            setEditStream({ node_id: node.node_id, inputs: [""], outputs: [""] });
                            setStreamOpen(true);
                        }}>
                        Stream
                    </Button>
                </Tooltip>
            </Stack>

            {filteredAdapters.length > 0 && (
                <Box mb={2}>
                    <Stack direction="row" alignItems="center" spacing={1} mb={1}>
                        <SettingsInputAntennaIcon sx={{ fontSize: 14, color: "text.disabled" }} />
                        <Typography variant="caption" fontWeight={700}
                            sx={{ color: "text.disabled", textTransform: "uppercase", letterSpacing: 1 }}>
                            DVB Adapters
                        </Typography>
                    </Stack>
                    <Stack direction="row" flexWrap="wrap" gap={1.5}>
                        {filteredAdapters.map(a => (
                            <AdapterCard key={a.id} adapter={a} nodeOnline={online}
                                onEdit={() => setAdaptersOpen(true)}
                                onDelete={() => {}} />
                        ))}
                    </Stack>
                </Box>
            )}

            {filteredStreams.length > 0 ? (
                <Box>
                    <Stack direction="row" alignItems="center" spacing={1} mb={1}>
                        <LiveTvIcon sx={{ fontSize: 14, color: "text.disabled" }} />
                        <Typography variant="caption" fontWeight={700}
                            sx={{ color: "text.disabled", textTransform: "uppercase", letterSpacing: 1 }}>
                            Streams
                        </Typography>
                        <Typography variant="caption" sx={{ color: "text.disabled" }}>
                            ({filteredStreams.length}{q ? ` of ${streams.length}` : ""})
                        </Typography>
                    </Stack>
                    <Stack direction="row" flexWrap="wrap" gap={1.5}>
                        {filteredStreams.map(s => (
                            <StreamCard key={s.id} stream={s} nodeOnline={online}
                                onEdit={(s) => {
                                    sendDataToServer({ op: "getStream", stream_id: String(s.id) }).then(res => {
                                        if (res.status === "OK") {
                                            setEditStream({
                                                node_id: node.node_id,
                                                stream_id: String(s.id),
                                                name: res.name,
                                                enable: res.enable,
                                                inputs: res.inputs?.length ? res.inputs : [""],
                                                outputs: res.outputs?.length ? res.outputs : [""],
                                            });
                                            setStreamOpen(true);
                                        }
                                    });
                                }}
                                onDelete={handleDeleteStream} />
                        ))}
                    </Stack>
                </Box>
            ) : !q ? (
                <Box sx={{ py: 2, color: "text.disabled", fontSize: 13 }}>
                    No streams. Click "Stream" to add one.
                </Box>
            ) : null}

            <Divider sx={{ mt: 3 }} />

            <AdaptersDialog open={adaptersOpen}
                onClose={() => { setAdaptersOpen(false); onRefresh(); }}
                nodeId={node.node_id} nodeName={node.name || node.address} />
            <InputEditDialog open={streamOpen} row={editStream}
                onClose={() => setStreamOpen(false)}
                onSaved={onRefresh} setEdges={() => {}} />
        </Box>
    );
}

function DashboardContent() {
    const [nodes, setNodes] = useState(null);
    const [searchOpen, setSearchOpen] = useState(false);
    const [searchQuery, setSearchQuery] = useState("");
    const searchRef = useRef(null);

    const load = useCallback(() => {
        sendDataToServer({ op: "getDashboardData" }).then(res => {
            if (res.status === "OK") setNodes(res.nodes || []);
        });
    }, []);

    useEffect(() => {
        load();
        const t = setInterval(load, 30000);
        return () => clearInterval(t);
    }, [load]);

    useEffect(() => {
        const handleKey = (e) => {
            if (e.key === "Escape" && searchOpen) {
                setSearchOpen(false);
                setSearchQuery("");
                return;
            }
            if ((e.key === "s" || e.key === "S") && !e.ctrlKey && !e.metaKey && !e.altKey) {
                const tag = document.activeElement?.tagName?.toLowerCase();
                if (tag === "input" || tag === "textarea" || document.activeElement?.isContentEditable) return;
                e.preventDefault();
                setSearchOpen(true);
                setTimeout(() => searchRef.current?.focus(), 50);
            }
        };
        window.addEventListener("keydown", handleKey);
        return () => window.removeEventListener("keydown", handleKey);
    }, [searchOpen]);

    if (nodes === null) {
        return <Box display="flex" justifyContent="center" pt={8}><CircularProgress /></Box>;
    }

    if (nodes.length === 0) {
        return (
            <Box textAlign="center" pt={8} color="text.disabled">
                <Typography variant="h6">No nodes configured</Typography>
                <Typography variant="body2" mt={1}>Go to Flow editor to add an Astra node.</Typography>
            </Box>
        );
    }

    return (
        <Container maxWidth="xl" sx={{ py: 3 }}>
            {searchOpen && (
                <Box mb={2}>
                    <TextField
                        inputRef={searchRef}
                        size="small"
                        fullWidth
                        placeholder="Search streams and adapters… (Esc to close)"
                        value={searchQuery}
                        onChange={e => setSearchQuery(e.target.value)}
                        InputProps={{
                            startAdornment: (
                                <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment>
                            ),
                            endAdornment: searchQuery && (
                                <InputAdornment position="end">
                                    <CloseIcon fontSize="small" sx={{ cursor: "pointer" }}
                                        onClick={() => setSearchQuery("")} />
                                </InputAdornment>
                            ),
                        }}
                    />
                </Box>
            )}
            {nodes.map(nd => (
                <NodeSection key={nd.node.id} nodeData={nd} onRefresh={load} searchQuery={searchQuery} />
            ))}
        </Container>
    );
}

export default function DashboardPage() {
    return <AuthGuard><DashboardContent /></AuthGuard>;
}
