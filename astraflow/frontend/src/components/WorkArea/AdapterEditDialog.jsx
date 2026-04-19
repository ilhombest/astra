import React, { useEffect, useState } from "react";
import {
    Button, Dialog, DialogActions, DialogContent, DialogTitle,
    FormControlLabel, MenuItem, Stack, Switch, TextField
} from "@mui/material";
import { sendDataToServer } from "utils/functions";

const DVB_TYPES = ["DVB-S", "DVB-S2", "DVB-T", "DVB-T2", "DVB-C", "DVB-C2"];

const defaultData = () => ({
    id: 0, name: "", adapter: 0, device: 0,
    dvb_type: "DVB-S2", mac: "", enabled: true
});

export default function AdapterEditDialog({ open, onClose, row, nodeId, onSaved }) {
    const [data, setData] = useState(defaultData());
    const isEdit = !!row?.id;

    useEffect(() => {
        if (open) {
            setData(row ? {
                id: row.id, name: row.name || "", adapter: row.adapter ?? 0,
                device: row.device ?? 0, dvb_type: row.dvb_type || "DVB-S2",
                mac: row.mac || "", enabled: row.enabled ?? true
            } : defaultData());
        }
    }, [open, row]);

    const set = (k, v) => setData(p => ({ ...p, [k]: v }));

    const handleSave = () => {
        sendDataToServer({
            op: "saveAdapter",
            node_id: nodeId,
            id: data.id,
            name: data.name,
            adapter: String(data.adapter),
            device: String(data.device),
            dvb_type: data.dvb_type,
            mac: data.mac,
            enabled: data.enabled ? "1" : "0"
        }).then(res => {
            if (res.status === "OK") { onSaved(); onClose(); }
            else alert(res.status);
        });
    };

    const handleDelete = () => {
        if (!window.confirm(`Delete adapter "${row?.name}"?`)) return;
        sendDataToServer({ op: "deleteAdapter", id: data.id }).then(res => {
            if (res.status === "OK") { onSaved(); onClose(); }
            else alert(res.status);
        });
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
            <DialogTitle>{isEdit ? "Edit Adapter" : "Add Adapter"}</DialogTitle>
            <DialogContent>
                <Stack spacing={2} sx={{ mt: 1 }}>
                    <TextField label="Name" value={data.name}
                        onChange={e => set("name", e.target.value)} fullWidth />
                    <Stack direction="row" spacing={2}>
                        <TextField label="Adapter №" type="number" value={data.adapter}
                            onChange={e => set("adapter", Number(e.target.value))}
                            inputProps={{ min: 0 }} fullWidth />
                        <TextField label="Device №" type="number" value={data.device}
                            onChange={e => set("device", Number(e.target.value))}
                            inputProps={{ min: 0 }} fullWidth />
                    </Stack>
                    <TextField select label="DVB Type" value={data.dvb_type}
                        onChange={e => set("dvb_type", e.target.value)} fullWidth>
                        {DVB_TYPES.map(t => <MenuItem key={t} value={t}>{t}</MenuItem>)}
                    </TextField>
                    <TextField label="MAC (optional)" value={data.mac}
                        onChange={e => set("mac", e.target.value)}
                        placeholder="00:11:22:33:44:55" fullWidth />
                    <FormControlLabel
                        control={<Switch checked={data.enabled}
                            onChange={e => set("enabled", e.target.checked)} />}
                        label="Enabled" />
                </Stack>
            </DialogContent>
            <DialogActions sx={{ justifyContent: "space-between" }}>
                <div>
                    {isEdit && <Button color="error" onClick={handleDelete}>Delete</Button>}
                </div>
                <Stack direction="row" spacing={1}>
                    <Button onClick={onClose}>Cancel</Button>
                    <Button variant="contained" onClick={handleSave}>Save</Button>
                </Stack>
            </DialogActions>
        </Dialog>
    );
}
