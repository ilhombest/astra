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

	fill := func(row *ClusterAdapter) {
		row.Name         = strings.TrimSpace(d["name"])
		row.DvbType      = strings.TrimSpace(d["dvb_type"])
		row.MAC          = strings.TrimSpace(d["mac"])
		row.Adapter      = int(toInt(d["adapter"]))
		row.Device       = int(toInt(d["device"]))
		row.Enabled      = d["enabled"] == "1"
		// tuning
		row.Frequency    = int(toInt(d["frequency"]))
		row.Polarization = strings.TrimSpace(d["polarization"])
		row.Symbolrate   = int(toInt(d["symbolrate"]))
		row.Lof1         = int(toInt(d["lof1"]))
		row.Lof2         = int(toInt(d["lof2"]))
		row.Slof         = int(toInt(d["slof"]))
		row.Bandwidth    = int(toInt(d["bandwidth"]))
		row.Modulation   = strings.TrimSpace(d["modulation"])
		// advanced
		row.BudgetMode   = d["budget_mode"] == "1"
		row.CaDelay      = int(toInt(d["ca_delay"]))
		row.ErrorTimeout = int(toInt(d["error_timeout"]))
		if row.DvbType == "" {
			row.DvbType = "DVB-S2"
		}
		if row.ErrorTimeout == 0 {
			row.ErrorTimeout = 120
		}
	}

	if id > 0 {
		var row ClusterAdapter
		if err := db.First(&row, id).Error; err != nil {
			out["status"] = "Adapter not found"
			return out
		}
		fill(&row)
		if err := db.Save(&row).Error; err != nil {
			out["status"] = err.Error()
			return out
		}
		out["row"] = row
	} else {
		row := ClusterAdapter{NodeID: nodeID}
		fill(&row)
		if row.Lof1 == 0 { row.Lof1 = 9750 }
		if row.Lof2 == 0 { row.Lof2 = 10600 }
		if row.Slof == 0 { row.Slof = 11700 }
		if row.Bandwidth == 0 { row.Bandwidth = 8 }
		if row.Modulation == "" { row.Modulation = "QAM256" }
		if row.Polarization == "" { row.Polarization = "H" }
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
