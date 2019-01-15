package main

import (
    "github.com/toravir/shellmgr/libs"
    "fmt"
    "log"
    "net"
    "net/http"
)


func main() {
        stype, saddr := shlmgr.ParseCmdLineArgs()
        fmt.Println("Listening on ", stype, saddr)

	l, err := net.Listen(stype, saddr)
	if err != nil {
                log.Fatal("Listen failed: %s\n", err)
	} else {
                shlmgr.RegisterSignalHandler(l)
                router := shlmgr.RegisterUrlRouters()
		err := http.Serve(l, router)
		if err != nil {
			panic(err)
		}
	}
}
