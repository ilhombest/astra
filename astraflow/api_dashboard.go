package main

import (
	"fmt"
	"strings"
)

type DashboardNode struct {
	Node     ClusterNode      `json:"node"`
	Streams  []ClusterStream  `json:"streams"`
	Adapters []ClusterAdapter `json:"adapters"`
}

func apiGetDashboardData(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	var nodes []ClusterNode
	if err := db.Order("id ASC").Find(&nodes).Error; err != nil {
		out["status"] = err.Error()
		return out
	}
	result := make([]DashboardNode, 0, len(nodes))
	for _, node := range nodes {
		var streams []ClusterStream
		db.Where("node_id = ?", node.NodeID).Order("name ASC").Find(&streams)
		var adapters []ClusterAdapter
		db.Where("node_id = ?", node.NodeID).Order("adapter, device").Find(&adapters)
		result = append(result, DashboardNode{
			Node:     node,
			Streams:  streams,
			Adapters: adapters,
		})
	}
	out["nodes"] = result
	return out
}

func apiGetStreamStatus(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	streamID := toUint(ctx.D["stream_id"])
	var stream ClusterStream
	if err := db.First(&stream, streamID).Error; err != nil {
		out["status"] = "stream not found"
		return out
	}
	var node ClusterNode
	db.Where("node_id = ?", stream.NodeID).First(&node)
	if node.ID == 0 {
		out["status"] = "node not found"
		return out
	}
	resp, err := node.Get("stream-status/" + stream.AstraID + "?t=0")
	if err != nil {
		out["offline"] = true
		return out
	}
	out["data"] = resp
	return out
}

func apiGetAdapterStatus(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	adapterID := toUint(ctx.D["adapter_id"])
	var adapter ClusterAdapter
	if err := db.First(&adapter, adapterID).Error; err != nil {
		out["status"] = "adapter not found"
		return out
	}
	var node ClusterNode
	db.Where("node_id = ?", adapter.NodeID).First(&node)
	if node.ID == 0 {
		out["status"] = "node not found"
		return out
	}
	path := fmt.Sprintf("adapter-status/%d/%d", adapter.Adapter, adapter.Device)
	resp, err := node.Get(path)
	if err != nil {
		out["offline"] = true
		return out
	}
	out["data"] = resp
	return out
}

func apiGetAllSessions(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	var nodes []ClusterNode
	db.Where("status = ?", "online").Find(&nodes)

	type NodeSessions struct {
		Node     ClusterNode `json:"node"`
		Sessions any         `json:"sessions"`
		Status   string      `json:"status"`
	}

	result := make([]NodeSessions, 0, len(nodes))
	for _, node := range nodes {
		resp, err := node.Control("sessions")
		ns := NodeSessions{Node: node}
		if err != nil {
			ns.Status = "offline"
			ns.Sessions = []any{}
		} else {
			ns.Status = "online"
			ns.Sessions = resp["sessions"]
		}
		result = append(result, ns)
	}
	out["nodes"] = result
	return out
}

func apiGetNodeLog(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	nodeID := strings.TrimSpace(ctx.D["node_id"])
	if nodeID == "" {
		out["status"] = "node_id required"
		return out
	}
	var node ClusterNode
	if err := db.Where("node_id = ?", nodeID).First(&node).Error; err != nil {
		out["status"] = "node not found"
		return out
	}
	resp, err := node.Get("log")
	if err != nil {
		out["lines"] = []any{}
		return out
	}
	out["lines"] = resp["lines"]
	return out
}
