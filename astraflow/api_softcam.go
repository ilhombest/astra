package main

import "strings"

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
