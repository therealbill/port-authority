package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/therealbill/port-authority/actions"
	"github.com/therealbill/port-authority/common"
	"github.com/zenazn/goji/web"
)

func APIGetPortFromInstance(c web.C, w http.ResponseWriter, r *http.Request) {
	id := c.URLParams["id"]
	resp := common.InfoResponse{Status: "data"}
	port, err := actions.GetPortFromInstance(id)
	stop := returnUnhandledError(err, &w)
	if stop {
		return
	}
	log.Printf("gat '%s' for port from call for '%s'", port, id)
	if len(port) > 0 {
		iport, err := strconv.Atoi(port)
		if err != nil {
			resp.Status = "Error"
			resp.StatusMessage = "Invalid port integer received from data store. Someone needs to investigate this RIGHT NOW."
			log.Printf("port: '%s'", port)
		} else {
			resp.Data = iport
			resp.StatusMessage = "Port found"
		}
	} else {
		resp.Data = 0
		resp.StatusMessage = "No Port found"
	}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}

func APIGetInstanceFromPort(c web.C, w http.ResponseWriter, r *http.Request) {
	port := c.URLParams["port"]
	iport, err := strconv.Atoi(port)
	resp := common.InfoResponse{Status: "data"}
	if err != nil {
		resp.Status = "Client Error"
		resp.StatusMessage = "Invalid port integer passed. Use a valid port number"
	} else {
		name, err := actions.GetInstanceFromPort(iport)
		stop := returnUnhandledError(err, &w)
		if stop {
			return
		}
		sname := string(name)
		if len(sname) > 0 {
			resp.Data = sname
			resp.StatusMessage = "Mapping found"
		} else {
			resp.Data = ""
			resp.StatusMessage = "No mapping found"
		}
	}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}

func APIGetOpenPort(c web.C, w http.ResponseWriter, r *http.Request) {
	id := c.URLParams["id"]
	port, err := actions.GetOpenPort(id)
	stop := returnUnhandledError(err, &w)
	if stop {
		return
	}
	resp := common.InfoResponse{Status: "data", StatusMessage: "open port found", Data: port}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}

func APIGetPortCapacity(c web.C, w http.ResponseWriter, r *http.Request) {
	count, err := actions.GetOpenPortCount()
	stop := returnUnhandledError(err, &w)
	if stop {
		return
	}
	resp := common.InfoResponse{Status: "data", StatusMessage: "success", Data: count}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}

func APIGetAvailableInventory(c web.C, w http.ResponseWriter, r *http.Request) {
	ports, err := actions.GetOpenPortList()
	stop := returnUnhandledError(err, &w)
	if stop {
		return
	}
	resp := common.InfoResponse{Status: "data", StatusMessage: "success", Data: ports}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}

func APIGetAssignedCount(c web.C, w http.ResponseWriter, r *http.Request) {
	count, err := actions.GetReservedPortCount()
	stop := returnUnhandledError(err, &w)
	if stop {
		return
	}
	resp := common.InfoResponse{Status: "data", StatusMessage: "success", Data: count}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}

func APIGetAssignedList(c web.C, w http.ResponseWriter, r *http.Request) {
	ports, err := actions.GetReservedPortList()
	stop := returnUnhandledError(err, &w)
	if stop {
		return
	}
	resp := common.InfoResponse{Status: "data", StatusMessage: "success", Data: ports}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}

func APIRemoveService(c web.C, w http.ResponseWriter, r *http.Request) {
	id := c.URLParams["id"]
	err := actions.RemoveService(id)
	stop := returnUnhandledError(err, &w)
	if stop {
		return
	}
	resp := common.InfoResponse{Status: "data", StatusMessage: "it's gone, man"}
	packed, _ := json.Marshal(resp)
	w.Write(packed)
}
