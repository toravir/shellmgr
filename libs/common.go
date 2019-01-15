package shlmgr

import (
	"fmt"
	"github.com/gorilla/mux"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var (
	router              *mux.Router = mux.NewRouter()
	DEFAULT_CMD_TIMEOUT             = 1000 //1 second
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

var allShells []*activeShell

func RegisterSignalHandler(l net.Listener) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		fmt.Printf("Caught signal %s: shutting down Gracefully", sig)
		l.Close()
		os.Exit(0)
	}(sigCh)
}

func RegisterUrlRouters() *mux.Router {
	registerExecCmdRoute()
	registerCreateShellRoute()
	registerListShellsRoute()
	return router
}
