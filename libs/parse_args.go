package shlmgr

import (
	"flag"
	"github.com/go-ini/ini"
	"github.com/rs/zerolog"
	"log"
	"os"
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
			log.Fatal("Cannot load Config file:!!", err)
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
				logger.Debug().Msgf("Invalid Cmd Timeout specified: %d, using Default(1s)", cmdTimeout)
				cmdTimeout = DEFAULT_CMD_TIMEOUT
			}
			err := spawnShell(shlId, shellExe, endPattern, cmdTimeout)
			if err != nil {
				logger.Debug().Int("ShellId", shlId).AnErr("Error", err).Msg("Error creating Bootup shell")
			}
		}
		logCfg := cfg.Section("shellmgr_logger")
		logLevelCfg := logCfg.Key("level").String()
		logDestCfg := logCfg.Key("destination").String()
		logLevel := DEFAULT_LOG_LEVEL
		if logLevelCfg != "" {
			var err error
			logLevel, err = zerolog.ParseLevel(logLevelCfg)
			if err != nil {
				//Invalid LoglevelCfg - so use default log level
				logLevel = DEFAULT_LOG_LEVEL
			}
		}
		logDest := os.Stdout
		switch logDestCfg {
		case "":
			fallthrough
		case "<stdout>":
			break
		case "<stderr>":
			logDest = os.Stderr
		default:
			fil, err := os.Create(logDestCfg)
			if err == nil {
				logDest = fil
			}
		}
		logger = zerolog.New(logDest).Level(logLevel).With().Timestamp().Logger()
	}
	return stype, saddr
}
