package shlmgr

import (
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var (
	router                  = mux.NewRouter()
	DEFAULT_CMD_TIMEOUT     = uint(1000) //1000 ms
	DEFAULT_LOG_LEVEL       = zerolog.ErrorLevel
	DEFAULT_LOG_DESTINATION = "stdout"   // can be "stderr"/"stdout"/fill path to a file
	DEFAULT_READ_BUFSIZE    = uint(1024) // in bytes
	logger                  = zerolog.New(os.Stdout).Level(zerolog.ErrorLevel).With().Timestamp().Logger()
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
	endPattern  string
	cmdTimeout  uint
	shellExe    string
	readBufSize uint
}

var allShells []*activeShell

func RegisterSignalHandler(l net.Listener) {
	logger.Debug().Msg("Registering Signal Handler...")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		logger.Debug().Msgf("Caught signal %s: shutting down Gracefully", sig)
		l.Close()
		os.Exit(0)
	}(sigCh)
}

func RegisterUrlRouters() *mux.Router {
	logger.Debug().Msg("Entering RegisterUrlRouters()")
	registerExecCmdRoute()
	registerCreateShellRoute()
	registerListShellsRoute()
	return router
}
