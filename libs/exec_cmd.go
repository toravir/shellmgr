package shlmgr

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		logger.Debug().Int("ShellId", shlId).Msg("Shell Exited or Not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	logger.Debug().Int("ShellId", shlId).Msg("Shell Found")

	type execReq struct {
		Cmd        string `json:"command"`
		EndPattern string `json:"terminatePattern,omitempty"`
		CmdTimeout uint   `json:"commandTimeout,omitempty"`
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

	logger.Debug().Int("ShellId", shlId).Str("Cmd", req.Cmd).Msg("Executing Cmd in Shell.")
	startTime := time.Now()
	var resp execResp
	reqShl.sin <- req.Cmd
	endCmd := false

	for {
		select {
		case out := <-reqShl.sout:
			resp.Output += string(out)
			if endPattern != "" {
				if strings.HasSuffix(resp.Output, endPattern) {
					logger.Debug().Str("Cmd", req.Cmd).Msg("EndPattern Found")
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
	elapsed := time.Now().Sub(startTime)
	logger.Debug().Str("Cmd", req.Cmd).
		Int("StdoutSize", len(resp.Output)).
		Int("StderrSize", len(resp.Error)).
		Int64("TookNs", elapsed.Nanoseconds()).
		Msg("Completed Command")

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}

func registerExecCmdRoute() {
	router.HandleFunc("/{shellId}/exec", executeCmd)
}
