package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"log"
        "flag"
        "os"
        "os/signal"
        "syscall"
)

var (
	router *mux.Router = mux.NewRouter()
)

type activeShell struct {
	sin     chan string
	sout    chan string
	serr    chan string
	exitCh  chan bool
	exitErr error

	shellId    int
	terminated bool
	cmdObj     *exec.Cmd

	//Input Params
	endPattern string
	cmdTimeout int
	shellExe   string
}

var allShells []activeShell

func executeCmd(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shlId, _ := strconv.Atoi(params["shellId"])

	var reqShl *activeShell

	reqShl = nil
	for _, shl := range allShells {
		if shl.shellId == shlId {
			reqShl = &shl
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

func spawnShell(shl *activeShell) error {
	cmd := exec.Command(shl.shellExe)

	shl.sin = make(chan string, 1)
	shl.sout = make(chan string, 1)
	shl.serr = make(chan string, 1)
	shl.exitCh = make(chan bool, 1)
	shl.cmdObj = cmd

	outp, _ := cmd.StdoutPipe()
	errp, _ := cmd.StderrPipe()
	inp, _ := cmd.StdinPipe()

	cmd.Start()

	go relayPipe2Chan(outp, shl.sout)
	go relayPipe2Chan(errp, shl.serr)
	go relayChan2Pipe(inp, shl.sin, shl.exitCh)
	go monitorShell(shl)

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

	var newShell activeShell
	shlId := 0
	for _, shl := range allShells {
		if shl.shellId > shlId {
			shlId = shl.shellId
		}
	}
	shlId++
	newShell.shellId = shlId
	newShell.shellExe = req.ShellExe
	newShell.endPattern = req.EndPattern
	newShell.cmdTimeout = req.CmdTimeout
	newShell.terminated = false

	allShells = append(allShells, newShell)
	err = spawnShell(&allShells[len(allShells)-1])
	fmt.Println("new Shell:", newShell)

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

func listShells (w http.ResponseWriter, r *http.Request) {
	type listResp struct {
		ShellId int    `json:"shellId"`
		Error   string `json:"error"`
                Status  string `json:"status"`
                ShellExe string `json:"shell"`
                EndPattern string `json:"terminatePattern"`
                CmdTimeout int `json:"cmdTimeout"`
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
            fmt.Printf("%+v\n", shlList[i])
            fmt.Printf("%+v\n", shl)
	}
	respStatus := http.StatusOK
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(respStatus)
	if err := json.NewEncoder(w).Encode(shlList); err != nil {
		panic(err)
	}
}

func registerSignalHandler (l net.Listener) {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
        go func(c chan os.Signal) {
                sig := <-c
                fmt.Printf("Caught signal %s: shutting down Gracefully", sig)
                l.Close()
                os.Exit(0)
        }(sigCh)
}


func main() {
        sockType := flag.String("socktype", "unix", "specify a socket type (tcp or unix)")
        sockAddr := flag.String("sockaddr", "", "specify addr to listen (/tmp/gw.sock or localhost:12345)")

        flag.Parse()

        if *sockAddr == "" {
            log.Fatal("Please specify socket addr...")
        }
        if *sockType != "unix" && *sockType != "tcp" {
            log.Fatal("Unix & TCP sockets are only supported")
        }
	router.HandleFunc("/newShell", createShell)
        router.HandleFunc("/listShells", listShells)
	router.HandleFunc("/{shellId}/exec", executeCmd)

        fmt.Println("Listening on ", *sockType, *sockAddr)
	l, err := net.Listen(*sockType, *sockAddr)

	if err != nil {
                log.Fatal("Listen failed: %s\n", err)
	} else {
                registerSignalHandler(l)
		err := http.Serve(l, router)
		if err != nil {
			panic(err)
		}
	}
}
