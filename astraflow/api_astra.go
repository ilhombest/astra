package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

// deployNodeToAstra builds a full config from DB and uploads it to astra-api.
// astra-api converts it to Lua (with dvb_tune blocks) and restarts Astra.
func deployNodeToAstra(node ClusterNode) error {
	// adapters
	var adapters []ClusterAdapter
	db.Where("node_id = ?", node.NodeID).Order("adapter, device").Find(&adapters)
	adapterMap := map[string]any{}
	for _, a := range adapters {
		if !a.Enabled {
			continue
		}
		key := fmt.Sprintf("%d", a.Adapter)
		adapterMap[key] = map[string]any{
			"adapter":      a.Adapter,
			"device":       a.Device,
			"dvb_type":     a.DvbType,
			"frequency":    a.Frequency,
			"polarization": a.Polarization,
			"symbolrate":   a.Symbolrate,
			"lof1":         a.Lof1,
			"lof2":         a.Lof2,
			"slof":         a.Slof,
			"bandwidth":    a.Bandwidth,
			"modulation":   a.Modulation,
		}
	}

	// newcamd servers
	var ncServers []NewcamdServer
	db.Where("node_id = ? AND enabled = ?", node.NodeID, true).Find(&ncServers)
	camsMap := map[string]any{}
	for _, nc := range ncServers {
		camID := fmt.Sprintf("cam_%d", nc.ID)
		camsMap[camID] = map[string]any{
			"type": "newcamd",
			"name": nc.Name,
			"host": nc.Host,
			"port": nc.Port,
			"user": nc.Username,
			"pass": nc.Password,
			"key":  nc.DESKey,
		}
	}

	// streams + ports
	var streams []ClusterStream
	db.Where("node_id = ?", node.NodeID).Find(&streams)
	streamMap := map[string]any{}
	for _, s := range streams {
		var inputPorts, outputPorts []ClusterPort
		db.Where("stream_id = ? AND direction = ?", s.ID, "input").Order("position").Find(&inputPorts)
		db.Where("stream_id = ? AND direction = ?", s.ID, "output").Order("position").Find(&outputPorts)
		inputs := make([]string, len(inputPorts))
		for i, p := range inputPorts {
			inputs[i] = p.Address
		}
		outputs := make([]string, len(outputPorts))
		for i, p := range outputPorts {
			outputs[i] = p.Address
		}
		entry := map[string]any{
			"id":     s.AstraID,
			"name":   s.Name,
			"enable": s.Enable,
			"type":   s.Type,
			"input":  inputs,
			"output": outputs,
		}
		if s.BissMode > 0 && s.BissKey != "" {
			entry["biss"] = s.BissKey
		}
		streamMap[s.AstraID] = entry
	}

	cfg := map[string]any{
		"adapters": adapterMap,
		"cams":     camsMap,
		"streams":  streamMap,
	}

	_, err := astraCommandJSON(node.Address, node.Auth, map[string]any{
		"cmd":    "upload",
		"config": cfg,
	})
	return err
}

func apiDeployToAstra(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	nodeID := strings.TrimSpace(ctx.D["node_id"])
	if nodeID == "" {
		out["status"] = "node_id required"
		return out
	}
	node := getNodeByNodeId(nodeID)
	if node == nil {
		out["status"] = "node not found"
		return out
	}
	if err := deployNodeToAstra(*node); err != nil {
		out["status"] = err.Error()
		return out
	}
	return out
}

func apiRestartNode(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	d := ctx.D

	nodeID := strings.TrimSpace(d["node_id"])
	if nodeID == "" {
		out["status"] = "node_id empty"
		return out
	}

	node := getNodeByNodeId(nodeID)
	if node == nil {
		out["status"] = "node not found"
		return out
	}

	if _, err := node.Control("restart"); err != nil {
		out["status"] = err.Error()
		return out
	}

	out["status"] = "OK"
	return out
}

func apiGetAstraConfig(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	d := ctx.D

	var node ClusterNode
	if err := db.
		Where("node_id = ?", d["node_id"]).
		First(&node).Error; err != nil {
		out["status"] = err.Error()
		return out
	}

	out["status"] = "OK"
	out["config"] = node.ConfigJSON
	return out
}

func apiSendConfigToAstra(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	d := ctx.D

	var node ClusterNode
	if err := db.
		Model(&ClusterNode{}).
		Clauses(clause.Returning{}).
		Where("node_id = ?", d["node_id"]).
		Updates(map[string]any{
			"config_json":       d["config"],
			"config_updated_at": time.Now(),
		}).
		Scan(&node).Error; err != nil {
		out["status"] = err.Error()
		return out
	}

	var cfg any
	if err := json.Unmarshal([]byte(d["config"]), &cfg); err != nil {
		out["status"] = "Invalid JSON: " + err.Error()
		return out
	}

	_, err := astraCommandJSON(node.Address, node.Auth, map[string]any{
		"cmd":    "upload",
		"config": cfg,
	})
	if err != nil {
		out["status"] = err.Error()
		return out
	}

	out["status"] = "OK"
	return out
}

func apiSystemInfo(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	d := ctx.D

	nodeID := d["node_id"]
	if nodeID == "" {
		out["status"] = "node_id empty"
		return out
	}
	node := getNodeByNodeId(nodeID)
	out, err := node.Control("sessions")
	if err != nil {
		out["status"] = err.Error()
		return out
	}
	systemStatus, err := node.Get("system-status")
	if err != nil {
		out["status"] = err.Error()
		return out
	}
	out["data"] = systemStatus
	out["status"] = "OK"
	return out
}

func apiStreamInfo(ctx *ApiCtx) map[string]any {
	out := ctx.Out
	d := ctx.D

	portID := strings.TrimSpace(d["port_id"])
	if portID == "" {
		out["status"] = "port_id empty"
		return out
	}

	var port ClusterPort
	if err := db.First(&port, portID).Error; err != nil {
		out["status"] = err.Error()
		return out
	}

	var stream ClusterStream
	if err := db.First(&stream, port.StreamID).Error; err != nil {
		out["status"] = err.Error()
		return out
	}

	var node ClusterNode
	if err := db.Where("node_id = ?", stream.NodeID).First(&node).Error; err != nil {
		out["status"] = err.Error()
		return out
	}

	resp, err := node.Get("stream-status/" + stream.AstraID + "?t=0")
	if err != nil {
		out["status"] = err.Error()
		return out
	}

	out["status"] = "OK"
	out["data"] = resp
	return out
}
