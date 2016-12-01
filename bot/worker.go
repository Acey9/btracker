package main

import (
	"fmt"
	"os"
	"time"
)

type Worker struct {
	Name     string
	count    int
	bots     map[string]*Bot
	addQueue chan *Bot
	delQueue chan *Bot
	startTS  int64
}

func NewWorker() *Worker {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	now := time.Now()
	worker := &Worker{hostname, 0, make(map[string]*Bot), make(chan *Bot), make(chan *Bot), now.Unix()}
	return worker
}

func (w *Worker) Handler() {
	go w.work()
	go w.heartBeat()
}

func (w *Worker) StartBot(b *Bot) {
	w.addQueue <- b
	bts.logger.Notice("Start <Bot:%s> %s", b.bid, b.cmdStr)
	b.cmd.Process.Wait()
	w.delQueue <- b
	bts.logger.Notice("Exit <Bot:%s> %s", b.bid, b.cmdStr)
}

func (w *Worker) StopBot(b *Bot) {
	//w.delQueue <- b
	b.Stop()
	bts.logger.Notice("Stop <Bot:%s> %s", b.bid, b.cmdStr)
}

func (w *Worker) DelBot(b *Bot) {
	w.delQueue <- b
	bts.logger.Notice("Del <Bot:%s> %s", b.bid, b.cmdStr)
}

func (w *Worker) work() {
	bts.logger.Notice("Start <work:%s>", w.Name)
	for {
		select {
		case start := <-w.addQueue:
			w.count++
			w.bots[start.bid] = start
			break
		case stop := <-w.delQueue:
			w.count--
			delete(w.bots, stop.bid)
			break
		}
	}
}

func (w *Worker) heartBeat() {
	bts.logger.Notice("Start <heartBeat:%s>", w.Name)
	key := fmt.Sprintf("%s/heartbeat/%s", bts.settings.ETCD.RootPath, w.Name)
	for {
		bts.logger.Info("Beat <worker:%s> %d", w.Name, w.count)

		sysinfo := NewSysInfo()
		info := fmt.Sprintf("%d\t%.2f\t%d\t%.2f\t%s\t%d\t%.2f\t%d", w.count, sysinfo.CPU[0], sysinfo.VMem.Free, sysinfo.VMem.UsedPercent, sysinfo.Load.String(), sysinfo.Disk.Free, sysinfo.Disk.UsedPercent, w.startTS)

		bts.etcdctl.SetWithTTL(key, info, false, time.Second*time.Duration(bts.settings.BotHeartbeatTTL))

		_sleep := bts.settings.BotHeartbeatTTL / 2
		time.Sleep(time.Second * time.Duration(_sleep))
	}
}
