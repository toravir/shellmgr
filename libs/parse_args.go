package shlmgr

import (
	"flag"
	"fmt"
	"github.com/go-ini/ini"
        "os"
        "log"
)

func ParseCmdLineArgs() (stype, saddr string) {
	sockType := flag.String("socktype", "unix", "specify a socket type (tcp or unix)")
	sockAddr := flag.String("sockaddr", "", "specify addr to listen (/tmp/gw.sock or localhost:12345)")
	config := flag.String("conf", "", "specify file which contains config for shell mgr")
	flag.Parse()

	stype = ""
	saddr = ""
	var cfg *ini.File = nil

	if *config == "" {
		stype = *sockType
		saddr = *sockAddr
	} else {
		var err error
		cfg, err = ini.Load(*config)
		if err != nil {
			fmt.Println("Cannot load Config file:!!", err)
			os.Exit(1)
		}
		globalCfg := cfg.Section("global")
		stype = globalCfg.Key("socket.type").String()
		saddr = globalCfg.Key("socket.address").String()
	}

	if saddr == "" {
		//Crash and burn
		log.Fatal("Please specify socket addr...")
	}

	if stype != "unix" && stype != "tcp" {
		//Crash and burn
		log.Fatal("Unix & TCP sockets are only supported")
	}

	if cfg != nil {
		shellsCfg := cfg.Section("bootup_shells")
		chSections := shellsCfg.ChildSections()
		for _, sh := range chSections {
			shlId, _ := sh.Key("shellid").Int()
			shellExe := sh.Key("shellexe").String()
			endPattern := sh.Key("terminatepattern").String()
			cmdTimeout, _ := sh.Key("cmdtimeout").Int()
			if cmdTimeout <= 0 {
				fmt.Println("Invalid Cmd Timeout specified:", cmdTimeout, ", using Default(1s)")
				cmdTimeout = DEFAULT_CMD_TIMEOUT
			}
			err := spawnShell(shlId, shellExe, endPattern, cmdTimeout)
			if err != nil {
				fmt.Println("Error creating bootup shell", sh, ", Error:", err)
			}
		}
	}
	return stype, saddr
}
