package shlmgr

import (
	"encoding/json"
	"fmt"
	"io"
        "os/exec"
        "net/http"
	"io/ioutil"
)

func relayPipe2Chan(pipe io.ReadCloser, out chan<- string) {
	for {
		lastRead := make([]byte, 100)
		n, _ := pipe.Read(lastRead)
		if n > 0 {
			out <- string(lastRead[:n])
		} else {
			fmt.Println("relayPipe2Chan Exiting...")
			return
		}
	}
}

func relayChan2Pipe(pipe io.WriteCloser, in <-chan string, exitCh <-chan bool) {
	for {
		select {
		case toWrite := <-in:
			fmt.Println("Wrote string:", string(toWrite))
			pipe.Write([]byte(toWrite))
		case <-exitCh:
			fmt.Println("relayChan2Pipe Exiting...")
			return
		}
	}
}

func monitorShell(shl *activeShell) {
	if shl.cmdObj != nil {
		shl.exitErr = shl.cmdObj.Wait()
		fmt.Println("monitorShell Exited...", shl.exitErr)
		shl.terminated = true
		shl.exitCh <- true
	}
}

func spawnShell(shlId int, shellExe string, endPattern string, cmdTimeout int) error {
	for _, v := range allShells {
		if v.shellId == shlId {
			return fmt.Errorf("Shell Id of %d is already used up..", shlId)
		}
	}
	newShell := activeShell{shellId: shlId,
		shellExe:   shellExe,
		endPattern: endPattern,
		cmdTimeout: cmdTimeout,
		terminated: false}

	allShells = append(allShells, &newShell)

	cmd := exec.Command(shellExe)

	newShell.sin = make(chan string, 1)
	newShell.sout = make(chan string, 1)
	newShell.serr = make(chan string, 1)
	newShell.exitCh = make(chan bool, 1)
	newShell.cmdObj = cmd

	outp, _ := cmd.StdoutPipe()
	errp, _ := cmd.StderrPipe()
	inp, _ := cmd.StdinPipe()

	err := cmd.Start()
	if err != nil {
		newShell.terminated = true
		newShell.exitErr = err
		return err
	}

	go relayPipe2Chan(outp, newShell.sout)
	go relayPipe2Chan(errp, newShell.serr)
	go relayChan2Pipe(inp, newShell.sin, newShell.exitCh)
	go monitorShell(&newShell)

	fmt.Println("created new Shell:", newShell)
	return nil
}

func createShell(w http.ResponseWriter, r *http.Request) {

	type createReq struct {
		ShellExe   string `json:"shell"`
		EndPattern string `json:"terminatePattern,omitempty"`
		CmdTimeout int    `json:"commandTimeout,omitempty"`
	}

	type createResp struct {
		ShellId int    `json:"shellId"`
		Error   string `json:"error"`
	}

	var req createReq
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
	//TODO - add input validations
	fmt.Println("new Shell:", req)

	shlId := 0
	for _, shl := range allShells {
		if shl.shellId > shlId {
			shlId = shl.shellId
		}
	}
	shlId++
	if req.CmdTimeout <= 0 {
		//If not specified or -ve, use default
		req.CmdTimeout = DEFAULT_CMD_TIMEOUT
	}

	err = spawnShell(shlId, req.ShellExe, req.EndPattern, req.CmdTimeout)

	var resp createResp
	resp.ShellId = shlId
	respStatus := http.StatusOK
	if err != nil {
		respStatus = http.StatusInternalServerError
		resp.Error = err.Error()
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(respStatus)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}

func registerCreateShellRoute() {
	router.HandleFunc("/newShell", createShell)
}
