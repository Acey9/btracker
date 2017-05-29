package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Worker struct {
	Name            string
	count           int
	bots            map[string]*Bot
	startQueue      chan *Bot
	delBotsMapQueue chan *Bot
	checkQueue      chan string
	stopQueue       chan string
	startTS         int64
	cntMutex        *sync.Mutex
}

func NewWorker() *Worker {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	now := time.Now()
	worker := &Worker{
		hostname,
		0,
		make(map[string]*Bot),
		make(chan *Bot),
		make(chan *Bot),
		make(chan string),
		make(chan string),
		now.Unix(),
		&sync.Mutex{},
	}
	return worker
}

func (w *Worker) Handler() {
	go w.work()
	go w.heartBeat()
}

func (w *Worker) botInit(cmdStr string) (*Bot, error) {
	bid := w.GetBotID(cmdStr)
	args, err := Split(cmdStr)
	if err != nil {
		bts.logger.Error("bot start failed:%s", err)
		return nil, err
	}

	cmdPath := fmt.Sprintf("%s/%s", bts.settings.BotBinPath, args[0])
	cmd := exec.Command(cmdPath, args[1:]...)

	err = cmd.Start()
	if err != nil {
		bts.logger.Error("bot start failed:%s", err)
		return nil, err
	}
	return NewBot(bid, cmdStr, cmd), nil
}

func (w *Worker) StartBot(cmdStr string) {
	b, err := w.botInit(cmdStr)
	if err != nil {
		return
	}
	w.startQueue <- b
	bts.logger.Notice("Start <Bot:%s> %s", b.bid, b.cmdStr)
	b.cmd.Process.Wait()
	w.delBotsMapQueue <- b
	bts.logger.Notice("Stoped <Bot:%s> %s", b.bid, b.cmdStr)
}

func (w *Worker) CheckBot(cmdStr string) {
	w.checkQueue <- cmdStr
}

func (w *Worker) StopBot(cmdStr string) {
	w.stopQueue <- cmdStr
}

func (w *Worker) GetBotID(cmdStr string) string {
	bid := fmt.Sprintf("%x", md5.Sum([]byte(cmdStr)))
	return bid
}

func (w *Worker) work() {
	bts.logger.Notice("Start <work:%s>", w.Name)
	for {
		select {
		case start := <-w.startQueue:
			w.cntMutex.Lock()
			defer w.cntMutex.Unlock()

			w.count++
			w.bots[start.bid] = start
		case del := <-w.delBotsMapQueue:
			w.cntMutex.Lock()
			defer w.cntMutex.Unlock()

			w.count--
			delete(w.bots, del.bid)
		case cmdStr := <-w.stopQueue:
			bid := w.GetBotID(cmdStr)

			w.cntMutex.Lock()
			defer w.cntMutex.Unlock()

			bot, ok := w.bots[bid]
			if ok {
				bot.Stop()
			}
		case cmdStr := <-w.checkQueue:
			bid := w.GetBotID(cmdStr)

			w.cntMutex.Lock()
			defer w.cntMutex.Unlock()

			_, ok := w.bots[bid]
			if !ok {
				go w.StartBot(cmdStr)
			}
		}
	}
}

func (w *Worker) heartBeat() {
	bts.logger.Notice("Start <heartBeat:%s>", w.Name)
	key := fmt.Sprintf("%s/heartbeat/%s", bts.settings.ETCD.RootPath, w.Name)
	for {
		bts.logger.Info("Beat <worker:%s> %d", w.Name, w.count)

		sysinfo := NewSysInfo()
		info := fmt.Sprintf("%d\t%.2f\t%d\t%.2f\t%s\t%d\t%.2f\t%d",
			w.count, sysinfo.CPU[0],
			sysinfo.VMem.Free,
			sysinfo.VMem.UsedPercent,
			sysinfo.Load.String(),
			sysinfo.Disk.Free,
			sysinfo.Disk.UsedPercent,
			w.startTS)

		bts.etcdctl.SetWithTTL(key, info, false, time.Second*time.Duration(bts.settings.BotHeartbeatTTL))

		_sleep := bts.settings.BotHeartbeatTTL / 2
		time.Sleep(time.Second * time.Duration(_sleep))
	}
}
