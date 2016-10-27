package main

import (
	"flag"
	"net/http"

	logger "github.com/jbrodriguez/mlog"
)

var (
	addr = flag.String("addr", ":80", "tcp address to listen to")
	maxSize = 1024 * 1024
)

func main() {
	flag.Parse()
	initLog()
	e := newEngine()
	e.init()
	err := http.ListenAndServe(*addr, e)
	if err != nil {
		logger.Error(err)
	}
}

func initLog() {
	logger.Start(logger.LevelInfo, "./info.log")
}
