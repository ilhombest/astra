import React, { useEffect, useState } from "react";
import {
    Accordion, AccordionDetails, AccordionSummary,
    Button, Dialog, DialogActions, DialogContent, DialogTitle,
    Divider, FormControlLabel, MenuItem, Stack, Switch,
    TextField, Typography
} from "@mui/material";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import { sendDataToServer } from "utils/functions";

const DVB_TYPES = ["DVB-S", "DVB-S2", "DVB-T", "DVB-T2", "DVB-C", "DVB-C2", "ATSC", "ISDB-T"];
const POLARIZATIONS = [
    { value: "H", label: "H — Horizontal" },
    { value: "V", label: "V — Vertical" },
    { value: "L", label: "L — Left circular" },
    { value: "R", label: "R — Right circular" },
];
const MODULATIONS = ["QAM64", "QAM128", "QAM256"];
const BANDWIDTHS = [6, 7, 8];

const isSatellite = (t) => t === "DVB-S" || t === "DVB-S2";
const isTerrestrial = (t) => t === "DVB-T" || t === "DVB-T2";
const isCable = (t) => t === "DVB-C" || t === "DVB-C2";

const defaults = () => ({
    id: 0, name: "", adapter: 0, device: 0,
    dvb_type: "DVB-S2", mac: "", enabled: true,
    // satellite
    frequency: 11000, polarization: "H", symbolrate: 27500,
    lof1: 9750, lof2: 10600, slof: 11700,
    // terrestrial
    bandwidth: 8,
    // cable
    modulation: "QAM256",
    // advanced
    budget_mode: false, ca_delay: 0, error_timeout: 120,
});

export default function AdapterEditDialog({ open, onClose, row, nodeId, onSaved }) {
    const [d, setD] = useState(defaults());
    const isEdit = !!row?.id;

    useEffect(() => {
        if (!open) return;
        if (row) {
            setD({
                id: row.id,           name: row.name || "",
                adapter: row.adapter ?? 0, device: row.device ?? 0,
                dvb_type: row.dvb_type || "DVB-S2", mac: row.mac || "",
                enabled: row.enabled ?? true,
                frequency: row.frequency || 11000,
                polarization: row.polarization || "H",
                symbolrate: row.symbolrate || 27500,
                lof1: row.lof1 || 9750, lof2: row.lof2 || 10600, slof: row.slof || 11700,
                bandwidth: row.bandwidth || 8,
                modulation: row.modulation || "QAM256",
                budget_mode: row.budget_mode ?? false,
                ca_delay: row.ca_delay ?? 0,
                error_timeout: row.error_timeout || 120,
            });
        } else {
            setD(defaults());
        }
    }, [open, row]);

    const set = (k, v) => setD(p => ({ ...p, [k]: v }));
    const num = (k, e) => set(k, Number(e.target.value));

    const handleSave = () => {
        const payload = {
            op: "saveAdapter", node_id: nodeId, id: d.id,
            name: d.name, adapter: String(d.adapter), device: String(d.device),
            dvb_type: d.dvb_type, mac: d.mac, enabled: d.enabled ? "1" : "0",
            frequency: String(d.frequency), polarization: d.polarization,
            symbolrate: String(d.symbolrate),
            lof1: String(d.lof1), lof2: String(d.lof2), slof: String(d.slof),
            bandwidth: String(d.bandwidth), modulation: d.modulation,
            budget_mode: d.budget_mode ? "1" : "0",
            ca_delay: String(d.ca_delay), error_timeout: String(d.error_timeout),
        };
        sendDataToServer(payload).then(res => {
            if (res.status === "OK") { onSaved(); onClose(); }
            else alert(res.status);
        });
    };

    const handleDelete = () => {
        if (!window.confirm(`Delete adapter "${row?.name}"?`)) return;
        sendDataToServer({ op: "deleteAdapter", id: d.id }).then(res => {
            if (res.status === "OK") { onSaved(); onClose(); }
            else alert(res.status);
        });
    };

    const sat  = isSatellite(d.dvb_type);
    const terr = isTerrestrial(d.dvb_type);
    const cab  = isCable(d.dvb_type);

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{isEdit ? "Edit Adapter" : "New Adapter"}</DialogTitle>

            <DialogContent dividers>
                <Stack spacing={2}>
                    {/* ── General ── */}
                    <Typography variant="overline" sx={{ color: "text.disabled" }}>General</Typography>
                    <Stack direction="row" spacing={2}>
                        <TextField label="Name" value={d.name}
                            onChange={e => set("name", e.target.value)} fullWidth />
                        <TextField select label="Type" value={d.dvb_type}
                            onChange={e => set("dvb_type", e.target.value)} sx={{ minWidth: 130 }}>
                            {DVB_TYPES.map(t => <MenuItem key={t} value={t}>{t}</MenuItem>)}
                        </TextField>
                    </Stack>
                    <FormControlLabel
                        control={<Switch checked={d.enabled} onChange={e => set("enabled", e.target.checked)} />}
                        label="Enabled" />

                    <Divider />
                    {/* ── Hardware ── */}
                    <Typography variant="overline" sx={{ color: "text.disabled" }}>Hardware</Typography>
                    <Stack direction="row" spacing={2}>
                        <TextField label="Adapter №" type="number" value={d.adapter}
                            onChange={e => num("adapter", e)} inputProps={{ min: 0 }} fullWidth />
                        <TextField label="Device №" type="number" value={d.device}
                            onChange={e => num("device", e)} inputProps={{ min: 0 }} fullWidth />
                        <TextField label="MAC (optional)" value={d.mac}
                            onChange={e => set("mac", e.target.value)}
                            placeholder="00:11:22:33:44:55" fullWidth />
                    </Stack>

                    {/* ── Satellite tuning ── */}
                    {sat && <>
                        <Divider />
                        <Typography variant="overline" sx={{ color: "text.disabled" }}>Satellite Tuning</Typography>
                        <Stack direction="row" spacing={2}>
                            <TextField label="Frequency (MHz)" type="number" value={d.frequency}
                                onChange={e => num("frequency", e)} fullWidth
                                helperText="e.g. 11000" />
                            <TextField select label="Polarization" value={d.polarization}
                                onChange={e => set("polarization", e.target.value)} fullWidth>
                                {POLARIZATIONS.map(p =>
                                    <MenuItem key={p.value} value={p.value}>{p.label}</MenuItem>)}
                            </TextField>
                            <TextField label="Symbol Rate (kBaud)" type="number" value={d.symbolrate}
                                onChange={e => num("symbolrate", e)} fullWidth
                                helperText="e.g. 27500" />
                        </Stack>

                        <Accordion disableGutters elevation={0}
                            sx={{ border: "1px solid", borderColor: "divider", borderRadius: "8px !important" }}>
                            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                                <Typography variant="body2">LNB Settings</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <Stack direction="row" spacing={2}>
                                    <TextField label="LOF1 (kHz)" type="number" value={d.lof1}
                                        onChange={e => num("lof1", e)} fullWidth helperText="Low band LO" />
                                    <TextField label="LOF2 (kHz)" type="number" value={d.lof2}
                                        onChange={e => num("lof2", e)} fullWidth helperText="High band LO" />
                                    <TextField label="SLOF (kHz)" type="number" value={d.slof}
                                        onChange={e => num("slof", e)} fullWidth helperText="Switch frequency" />
                                </Stack>
                            </AccordionDetails>
                        </Accordion>
                    </>}

                    {/* ── Terrestrial tuning ── */}
                    {terr && <>
                        <Divider />
                        <Typography variant="overline" sx={{ color: "text.disabled" }}>Terrestrial Tuning</Typography>
                        <Stack direction="row" spacing={2}>
                            <TextField label="Frequency (MHz)" type="number" value={d.frequency}
                                onChange={e => num("frequency", e)} fullWidth />
                            <TextField select label="Bandwidth (MHz)" value={d.bandwidth}
                                onChange={e => num("bandwidth", e)} fullWidth>
                                {BANDWIDTHS.map(b => <MenuItem key={b} value={b}>{b} MHz</MenuItem>)}
                            </TextField>
                        </Stack>
                    </>}

                    {/* ── Cable tuning ── */}
                    {cab && <>
                        <Divider />
                        <Typography variant="overline" sx={{ color: "text.disabled" }}>Cable Tuning</Typography>
                        <Stack direction="row" spacing={2}>
                            <TextField label="Frequency (MHz)" type="number" value={d.frequency}
                                onChange={e => num("frequency", e)} fullWidth />
                            <TextField label="Symbol Rate (kBaud)" type="number" value={d.symbolrate}
                                onChange={e => num("symbolrate", e)} fullWidth />
                            <TextField select label="Modulation" value={d.modulation}
                                onChange={e => set("modulation", e.target.value)} fullWidth>
                                {MODULATIONS.map(m => <MenuItem key={m} value={m}>{m}</MenuItem>)}
                            </TextField>
                        </Stack>
                    </>}

                    {/* ── Advanced ── */}
                    <Accordion disableGutters elevation={0}
                        sx={{ border: "1px solid", borderColor: "divider", borderRadius: "8px !important" }}>
                        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                            <Typography variant="body2">Advanced</Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Stack spacing={2}>
                                <FormControlLabel
                                    control={<Switch checked={d.budget_mode}
                                        onChange={e => set("budget_mode", e.target.checked)} />}
                                    label="Budget Mode (disable hardware PID filtering)" />
                                <Stack direction="row" spacing={2}>
                                    <TextField label="CA Delay (ms)" type="number" value={d.ca_delay}
                                        onChange={e => num("ca_delay", e)} fullWidth
                                        helperText="Delay between CA packets" />
                                    <TextField label="Error Timeout (s)" type="number" value={d.error_timeout}
                                        onChange={e => num("error_timeout", e)} fullWidth
                                        helperText="Check DVB errors interval" />
                                </Stack>
                            </Stack>
                        </AccordionDetails>
                    </Accordion>
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
