package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Worker struct {
	Name     string
	count    int
	bots     map[string]*Bot
	addQueue chan *Bot
	delQueue chan *Bot
}

func NewWorker() *Worker {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	worker := &Worker{hostname, 0, make(map[string]*Bot), make(chan *Bot), make(chan *Bot)}
	return worker
}

func (w *Worker) Handler() {
	go w.work()
	go w.heartbeat()
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

func (w *Worker) heartbeat() {
	bts.logger.Notice("Start <heartBeat:%s>", w.Name)
	key := fmt.Sprintf("%s/heartbeat/%s", bts.settings.ETCD.RootPath, w.Name)
	for {
		bts.logger.Info("Beat <worker:%s> %d", w.Name, w.count)
		bts.etcdctl.SetWithTTL(key, strconv.Itoa(w.count), false, time.Second*12)
		time.Sleep(time.Second * 8)
	}
}
