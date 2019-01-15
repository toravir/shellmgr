package shlmgr

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
        "net/http"
        "io"
        "io/ioutil"
        "strings"
        "time"
	"strconv"
)

func executeCmd(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shlId, _ := strconv.Atoi(params["shellId"])

	var reqShl *activeShell

	reqShl = nil
	for _, shl := range allShells {
		if shl.shellId == shlId {
			reqShl = shl
			break
		}
	}
	if reqShl == nil || reqShl.terminated {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fmt.Println("exec: shl", reqShl)

	type execReq struct {
		Cmd        string `json:"command"`
		EndPattern string `json:"terminatePattern,omitempty"`
		CmdTimeout int    `json:"commanTimeout,omitempty"`
	}

	type execResp struct {
		Output string `json:"output"`
		Error  string `json:"error"`
	}

	var req execReq
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	endPattern := reqShl.endPattern
	if req.EndPattern != "" {
		endPattern = req.EndPattern
	}
	cmdTimeout := reqShl.cmdTimeout
	if req.CmdTimeout > 0 {
		cmdTimeout = req.CmdTimeout
	}

	fmt.Println("exec: req", req)
	var resp execResp
	reqShl.sin <- req.Cmd
	endCmd := false

	for {
		fmt.Println("exec: resp", resp)
		select {
		case out := <-reqShl.sout:
			resp.Output += string(out)
			fmt.Println("current resp.Output:", resp.Output)
			if endPattern != "" {
				fmt.Println("Looking for endPattern:", endPattern)
				if strings.HasSuffix(resp.Output, endPattern) {
					fmt.Println("endPattern FOUND!!")
					endCmd = true
					break
				}
			}
		case err := <-reqShl.serr:
			resp.Error += string(err)
		case <-time.After(time.Duration(cmdTimeout) * time.Millisecond):
			resp.Error += "%%CMD Timed out !!"
			endCmd = true
			break
		}
		if endCmd {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}

func registerExecCmdRoute() {
	router.HandleFunc("/{shellId}/exec", executeCmd)
}
