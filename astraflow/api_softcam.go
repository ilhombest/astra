package main

import (
	"strings"
)

func apiGetSoftcam(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	var nodes []ClusterNode
	db.Order("id ASC").Find(&nodes)

	type NodeStreams struct {
		Node    ClusterNode    `json:"node"`
		Streams []ClusterStream `json:"streams"`
	}

	result := make([]NodeStreams, 0, len(nodes))
	for _, node := range nodes {
		var streams []ClusterStream
		db.Where("node_id = ?", node.NodeID).Order("name ASC").Find(&streams)
		result = append(result, NodeStreams{Node: node, Streams: streams})
	}
	out["nodes"] = result
	return out
}

func apiSaveSoftcam(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	id := toUint(ctx.D["stream_id"])
	if id == 0 {
		out["status"] = "stream_id required"
		return out
	}
	var stream ClusterStream
	if err := db.First(&stream, id).Error; err != nil {
		out["status"] = "stream not found"
		return out
	}
	stream.BissMode = int(toInt(ctx.D["biss_mode"]))
	stream.BissKey = strings.ToUpper(strings.TrimSpace(ctx.D["biss_key"]))
	if err := db.Save(&stream).Error; err != nil {
		out["status"] = err.Error()
		return out
	}
	out["row"] = stream
	return out
}

// ── NewCamd ──────────────────────────────────────────────────────────────────

func apiGetNewcamd(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	var nodes []ClusterNode
	db.Order("id ASC").Find(&nodes)

	type NodeNewcamd struct {
		Node    ClusterNode      `json:"node"`
		Servers []NewcamdServer  `json:"servers"`
	}

	result := make([]NodeNewcamd, 0, len(nodes))
	for _, node := range nodes {
		var servers []NewcamdServer
		db.Where("node_id = ?", node.NodeID).Order("name ASC").Find(&servers)
		result = append(result, NodeNewcamd{Node: node, Servers: servers})
	}
	out["nodes"] = result
	return out
}

func apiSaveNewcamd(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	d := ctx.D

	nodeID := strings.TrimSpace(d["node_id"])
	if nodeID == "" {
		out["status"] = "node_id required"
		return out
	}

	fill := func(s *NewcamdServer) {
		s.Name     = strings.TrimSpace(d["name"])
		s.Host     = strings.TrimSpace(d["host"])
		s.Port     = int(toInt(d["port"]))
		s.Username = strings.TrimSpace(d["username"])
		s.Password = strings.TrimSpace(d["password"])
		s.DESKey   = strings.ToUpper(strings.TrimSpace(d["des_key"]))
		s.Enabled  = d["enabled"] == "1"
		if s.Port == 0 {
			s.Port = 2222
		}
	}

	id := toUint(d["id"])
	if id > 0 {
		var row NewcamdServer
		if err := db.First(&row, id).Error; err != nil {
			out["status"] = "server not found"
			return out
		}
		fill(&row)
		if err := db.Save(&row).Error; err != nil {
			out["status"] = err.Error()
			return out
		}
		out["row"] = row
	} else {
		row := NewcamdServer{NodeID: nodeID}
		fill(&row)
		if err := db.Create(&row).Error; err != nil {
			out["status"] = err.Error()
			return out
		}
		out["row"] = row
	}
	return out
}

func apiDeleteNewcamd(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	id := toUint(ctx.D["id"])
	if id == 0 {
		out["status"] = "id required"
		return out
	}
	if err := db.Delete(&NewcamdServer{}, id).Error; err != nil {
		out["status"] = err.Error()
		return out
	}
	return out
}
