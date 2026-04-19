import React, { useCallback, useEffect, useState } from "react";
import {
    Box, Button, Chip, Dialog, DialogActions, DialogContent,
    DialogTitle, IconButton, Stack, Table, TableBody, TableCell,
    TableHead, TableRow, Typography
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import { sendDataToServer } from "utils/functions";
import AdapterEditDialog from "./AdapterEditDialog";

export default function AdaptersDialog({ open, onClose, nodeId, nodeName }) {
    const [rows, setRows] = useState([]);
    const [editOpen, setEditOpen] = useState(false);
    const [editRow, setEditRow] = useState(null);

    const load = useCallback(() => {
        if (!nodeId) return;
        sendDataToServer({ op: "getAdapters", node_id: nodeId }).then(res => {
            if (res.status === "OK") setRows(res.rows || []);
        });
    }, [nodeId]);

    useEffect(() => { if (open) load(); }, [open, load]);

    const handleAdd = () => { setEditRow(null); setEditOpen(true); };
    const handleEdit = (row) => { setEditRow(row); setEditOpen(true); };

    return (
        <>
            <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
                <DialogTitle>
                    <Stack direction="row" justifyContent="space-between" alignItems="center">
                        <Typography variant="h6">
                            Adapters — {nodeName}
                        </Typography>
                        <Button startIcon={<AddIcon />} variant="contained"
                            size="small" onClick={handleAdd}>
                            Add
                        </Button>
                    </Stack>
                </DialogTitle>

                <DialogContent>
                    {rows.length === 0 ? (
                        <Box sx={{ py: 4, textAlign: "center", color: "text.secondary" }}>
                            No adapters. Click "Add" to add a DVB adapter.
                        </Box>
                    ) : (
                        <Table size="small">
                            <TableHead>
                                <TableRow>
                                    <TableCell>Name</TableCell>
                                    <TableCell>Adapter / Device</TableCell>
                                    <TableCell>Type</TableCell>
                                    <TableCell>Tuning</TableCell>
                                    <TableCell>Status</TableCell>
                                    <TableCell align="right" />
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {rows.map(row => {
                                    const sat  = row.dvb_type === "DVB-S"  || row.dvb_type === "DVB-S2";
                                    const terr = row.dvb_type === "DVB-T"  || row.dvb_type === "DVB-T2";
                                    const cab  = row.dvb_type === "DVB-C"  || row.dvb_type === "DVB-C2";
                                    const tuning = sat
                                        ? `${row.frequency} MHz ${row.polarization || ""} ${row.symbolrate ? row.symbolrate + " kBd" : ""}`
                                        : terr
                                        ? `${row.frequency} MHz / ${row.bandwidth} MHz BW`
                                        : cab
                                        ? `${row.frequency} MHz ${row.modulation || ""}`
                                        : "—";
                                    return (
                                    <TableRow key={row.id} hover>
                                        <TableCell>{row.name || "—"}</TableCell>
                                        <TableCell>{row.adapter} / {row.device}</TableCell>
                                        <TableCell>
                                            <Chip label={row.dvb_type} size="small"
                                                color="primary" variant="outlined" />
                                        </TableCell>
                                        <TableCell sx={{ fontSize: 12, color: "text.secondary" }}>
                                            {tuning}
                                        </TableCell>
                                        <TableCell>
                                            <Chip label={row.enabled ? "on" : "off"} size="small"
                                                color={row.enabled ? "success" : "default"} />
                                        </TableCell>
                                        <TableCell align="right">
                                            <IconButton size="small" onClick={() => handleEdit(row)}>
                                                <EditIcon fontSize="inherit" />
                                            </IconButton>
                                        </TableCell>
                                    </TableRow>
                                    );
                                })}
                            </TableBody>
                        </Table>
                    )}
                </DialogContent>

                <DialogActions>
                    <Button onClick={onClose}>Close</Button>
                </DialogActions>
            </Dialog>

            <AdapterEditDialog
                open={editOpen}
                onClose={() => setEditOpen(false)}
                row={editRow}
                nodeId={nodeId}
                onSaved={load}
            />
        </>
    );
}
