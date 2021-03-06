package shlmgr

import (
	"encoding/json"
	"net/http"
)

func listShells(w http.ResponseWriter, r *http.Request) {
	type listResp struct {
		ShellId     int    `json:"shellId"`
		Error       string `json:"error"`
		Status      string `json:"status"`
		ShellExe    string `json:"shell"`
		EndPattern  string `json:"terminatePattern"`
		CmdTimeout  uint   `json:"cmdTimeout"`
		ReadBufSize uint   `json:"readBufSize"`
	}

	shlList := make([]listResp, len(allShells))
	for i, shl := range allShells {
		shlList[i].ShellId = shl.shellId
		if shl.exitErr != nil {
			shlList[i].Error = shl.exitErr.Error()
		}
		shlList[i].Status = "running"
		if shl.terminated {
			shlList[i].Status = "Exited"
		}
		shlList[i].ShellExe = shl.shellExe
		shlList[i].EndPattern = shl.endPattern
		shlList[i].CmdTimeout = shl.cmdTimeout
		shlList[i].ReadBufSize = shl.readBufSize
	}
	respStatus := http.StatusOK
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(respStatus)
	if err := json.NewEncoder(w).Encode(shlList); err != nil {
		panic(err)
	}
}

func registerListShellsRoute() {
	router.HandleFunc("/listShells", listShells)
}
