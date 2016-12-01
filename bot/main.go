package main

import (
	"flag"
	"fmt"
	"github.com/astaxie/beego/logs"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var bts BotServer

type BotServer struct {
	Debug        bool
	settingsFile string
	settings     *BotSettings
	logger       *logs.BeeLogger
	worker       *Worker
	manager      *Manager
	etcdctl      *ETCDCTL
}

func main() {
	initBotServer()
	sayHi()
	go bts.worker.Handler()
	go bts.manager.Handler()

	sig := make(chan os.Signal)
	//signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

forever:
	for {
		select {
		case <-sig:
			bts.manager.StopAllBot()
			fmt.Println("Interrupt signal recevied, stopping")
			break forever
		}
	}
}

func sayHi() {
	bts.logger.Info("%s : Netlab botnet track System - Bot Worker", bts.settings.Title)
}

func initBotServer() {
	bts.settings = &BotSettings{}
	err := NewSettings(bts.settingsFile, bts.settings)
	if err != nil {
		fmt.Printf("%s is not a valid toml config file\n", bts.settingsFile)
		fmt.Println(err)
		os.Exit(1)
	}
	initLogger()

	es := bts.settings.ETCD
	etcdctl, err := NewETCDCTL(es.Endpoints, time.Duration(es.Timeout)*time.Second, es.Username, es.Password)
	if err != nil {
		bts.logger.Error("Create etcd client failed: %s", err)
		os.Exit(1)
	}
	bts.etcdctl = etcdctl

	if bts.Debug {
		bts.settings.ETCD.RootPath = "/btracker/debug"
	}
	fmt.Println(bts.settings.ETCD.RootPath)

	bts.worker = NewWorker()
	bts.manager = NewManager()
}

func initLogger() {
	bts.logger = logs.NewLogger(10000)
	if bts.settings.Log.Stdout {
		bts.logger.SetLogger("console", "")
	}
	if bts.settings.Log.Path != "" {
		cfg := fmt.Sprintf(`{"filename":"%s"}`, bts.settings.Log.Path)
		bts.logger.SetLogger("file", cfg)
	}
	bts.logger.SetLevel(bts.settings.Log.BeeLevel())
	bts.logger.Async()
}

func optParse() {
	flag.StringVar(&bts.settingsFile, "c", "../../etc/bot.debug.conf", "Look for bot config file in this directory")
	flag.BoolVar(&bts.Debug, "d", false, "Only debug")
	flag.Parse()
}

func init() {
	optParse()
	runtime.GOMAXPROCS(runtime.NumCPU())
}
