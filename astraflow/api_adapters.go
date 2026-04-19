package main

import (
	"strings"
)

func apiGetAdapters(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	nodeID := strings.TrimSpace(ctx.D["node_id"])
	if nodeID == "" {
		out["status"] = "node_id required"
		return out
	}
	var rows []ClusterAdapter
	if err := db.Where("node_id = ?", nodeID).Order("adapter, device").Find(&rows).Error; err != nil {
		out["status"] = err.Error()
		return out
	}
	out["rows"] = rows
	return out
}

func apiSaveAdapter(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	d := ctx.D

	nodeID := strings.TrimSpace(d["node_id"])
	if nodeID == "" {
		out["status"] = "node_id required"
		return out
	}

	id := toUint(d["id"])
	name := strings.TrimSpace(d["name"])
	dvbType := strings.TrimSpace(d["dvb_type"])
	mac := strings.TrimSpace(d["mac"])
	adapter := int(toInt(d["adapter"]))
	device := int(toInt(d["device"]))
	enabled := d["enabled"] == "1"

	if dvbType == "" {
		dvbType = "DVB-S2"
	}

	if id > 0 {
		var row ClusterAdapter
		if err := db.First(&row, id).Error; err != nil {
			out["status"] = "Adapter not found"
			return out
		}
		row.Name = name
		row.DvbType = dvbType
		row.MAC = mac
		row.Adapter = adapter
		row.Device = device
		row.Enabled = enabled
		if err := db.Save(&row).Error; err != nil {
			out["status"] = err.Error()
			return out
		}
		out["row"] = row
	} else {
		row := ClusterAdapter{
			NodeID:  nodeID,
			Name:    name,
			DvbType: dvbType,
			MAC:     mac,
			Adapter: adapter,
			Device:  device,
			Enabled: enabled,
		}
		if err := db.Create(&row).Error; err != nil {
			out["status"] = err.Error()
			return out
		}
		out["row"] = row
	}
	return out
}

func apiDeleteAdapter(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	id := toUint(ctx.D["id"])
	if id == 0 {
		out["status"] = "id required"
		return out
	}
	if err := db.Delete(&ClusterAdapter{}, id).Error; err != nil {
		out["status"] = err.Error()
		return out
	}
	return out
}
