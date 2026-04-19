import React, { useCallback, useEffect, useState } from "react";
import {
    Box, Chip, CircularProgress, Container, Stack,
    Table, TableBody, TableCell, TableHead, TableRow,
    Typography
} from "@mui/material";
import PeopleIcon from "@mui/icons-material/People";
import AuthGuard from "components/Auth/AuthGuard";
import { sendDataToServer } from "utils/functions";

function formatUptime(sec) {
    if (!sec) return "—";
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    if (h > 0) return `${h}h ${m}m`;
    if (m > 0) return `${m}m ${s}s`;
    return `${s}s`;
}

function NodeSessionsBlock({ nodeData }) {
    const { node, sessions, status } = nodeData;
    const list = Array.isArray(sessions) ? sessions : [];
    const online = status === "online";

    return (
        <Box mb={4}>
            <Stack direction="row" spacing={1.5} alignItems="center" mb={1.5}>
                <Chip size="small" label={online ? "online" : "offline"}
                    color={online ? "success" : "default"}
                    sx={{ fontWeight: 700, fontSize: 10 }} />
                <Typography variant="subtitle1" fontWeight={800}>
                    {node.name || node.address}
                </Typography>
                <Chip size="small" label={`${list.length} sessions`}
                    icon={<PeopleIcon sx={{ fontSize: "12px !important" }} />}
                    variant="outlined" sx={{ fontSize: 11 }} />
            </Stack>

            {list.length === 0 ? (
                <Typography variant="body2" sx={{ color: "text.disabled", pl: 1 }}>
                    No active sessions
                </Typography>
            ) : (
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>#</TableCell>
                            <TableCell>IP Address</TableCell>
                            <TableCell>Path / Channel</TableCell>
                            <TableCell>Duration</TableCell>
                            <TableCell align="right">Bitrate</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {list.map((s, i) => (
                            <TableRow key={s.id || i} hover>
                                <TableCell sx={{ color: "text.disabled" }}>{i + 1}</TableCell>
                                <TableCell sx={{ fontFamily: "monospace", fontSize: 12 }}>
                                    {s.addr || "—"}
                                </TableCell>
                                <TableCell sx={{ fontSize: 12 }}>
                                    {s.path || s.channel || "—"}
                                </TableCell>
                                <TableCell sx={{ fontSize: 12 }}>
                                    {s.st ? formatUptime(Math.floor(Date.now() / 1000) - s.st) : "—"}
                                </TableCell>
                                <TableCell align="right" sx={{ fontSize: 12 }}>
                                    {s.bitrate ? `${s.bitrate} kb/s` : "—"}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}
        </Box>
    );
}

function SessionsContent() {
    const [data, setData] = useState(null);

    const load = useCallback(() => {
        sendDataToServer({ op: "getAllSessions" }).then(res => {
            if (res.status === "OK") setData(res.nodes || []);
        });
    }, []);

    useEffect(() => {
        load();
        const t = setInterval(load, 5000);
        return () => clearInterval(t);
    }, [load]);

    if (data === null) {
        return <Box display="flex" justifyContent="center" pt={8}><CircularProgress /></Box>;
    }

    const total = data.reduce((acc, n) => acc + (Array.isArray(n.sessions) ? n.sessions.length : 0), 0);

    return (
        <Container maxWidth="xl" sx={{ py: 3 }}>
            <Stack direction="row" spacing={2} alignItems="center" mb={3}>
                <PeopleIcon />
                <Typography variant="h6" fontWeight={700}>Sessions</Typography>
                <Chip label={`${total} total`} size="small" color={total > 0 ? "primary" : "default"} />
            </Stack>
            {data.length === 0 ? (
                <Typography color="text.disabled">No online nodes</Typography>
            ) : (
                data.map(nd => (
                    <NodeSessionsBlock key={nd.node.id} nodeData={nd} />
                ))
            )}
        </Container>
    );
}

export default function SessionsPage() {
    return <AuthGuard><SessionsContent /></AuthGuard>;
}
