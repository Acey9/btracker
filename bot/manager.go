package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os/exec"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/client"
)

type Manager struct {
	Name string
}

func NewManager() *Manager {
	return &Manager{bts.worker.Name}
}

func (m *Manager) Handler() {
	resp, err := m.GetConfig()
	if err != nil {
		bts.logger.Error("get config failed:%s", err)
		return
	}
	if resp != "" {
		bts.logger.Debug("bot config:%s", resp)
		cmds := strings.Split(resp, ";")
		for _, cmd := range cmds {
			go m.botStart(cmd)
		}
	}

	//go m.monitor()

	//TODO very dangerous interface
	go m.watchCmd()

	go m.watchConfig()
}

func (m *Manager) watchCmd() {
	key := bts.settings.ETCD.RootPath + "/cmds"
	bts.logger.Debug(key)
	bts.etcdctl.WatchForever(key, m.cmdChange, false)
}

func (m *Manager) cmdChange(resp *etcd.Response, err error) {
	if err != nil {
		return
	}
	if resp.Node.Value == "" || resp.Node.Value == resp.PrevNode.Value {
		return
	}
	bts.logger.Notice("cmd changed.")
	tmp := strings.Split(resp.Node.Value, ";")
	for _, cmd := range tmp[1:] {
		go m.execCommand(cmd)
	}

}

func (m *Manager) execCommand(cmdStr string) {
	args, err := Split(cmdStr)
	if err != nil {
		bts.logger.Error("Split command failed:%s", err)
		return
	}
	bts.logger.Debug("exec: %s", cmdStr)

	var out bytes.Buffer

	//cmdPath := fmt.Sprintf("./bin/%s", args[0])
	//cmd := exec.Command(cmdPath, args[1:]...)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		bts.logger.Error("start command failed:%s", err)
		return
	}
	bts.logger.Info("%q\n", out.String())
}

func (m *Manager) watchConfig() {
	key := bts.settings.ETCD.RootPath + "/conf/" + m.Name
	bts.etcdctl.WatchForever(key, m.configChange, false)
}

func (m *Manager) configChange(resp *etcd.Response, err error) {
	if err != nil {
		return
	}
	if resp.Node.Value == "" || resp.Node.Value == resp.PrevNode.Value {
		return
	}
	bts.logger.Notice("config changed.")
	adds := Difference(strings.Split(resp.Node.Value, ";"), strings.Split(resp.PrevNode.Value, ";"))
	for _, cmdStr := range adds {
		go m.botStart(cmdStr)
	}

	dels := Difference(strings.Split(resp.PrevNode.Value, ";"), strings.Split(resp.Node.Value, ";"))
	for _, cmdStr := range dels {
		go m.botStop(cmdStr)
	}
}

func (m *Manager) botStop(cmdStr string) {
	bid := m.getBotID(cmdStr)
	bot, ok := bts.worker.bots[bid]
	if ok {
		bts.worker.StopBot(bot)
	}
}

func (m *Manager) getBotID(cmdStr string) string {
	bid := fmt.Sprintf("%x", md5.Sum([]byte(cmdStr)))
	return bid
}

func (m *Manager) botStart(cmdStr string) {
	bid := m.getBotID(cmdStr)
	args, err := Split(cmdStr)
	if err != nil {
		bts.logger.Error("bot start failed:%s", err)
		return
	}

	cmdPath := fmt.Sprintf("%s/%s", bts.settings.BotBinPath, args[0])
	cmd := exec.Command(cmdPath, args[1:]...)

	err = cmd.Start()
	if err != nil {
		bts.logger.Error("bot start failed:%s", err)
		return
	}
	bot := NewBot(bid, cmdStr, cmd)
	bts.worker.StartBot(bot)
}

func (m *Manager) getFromETCD(key string) (string, error) {
	for {
		resp, err := bts.etcdctl.Get(key)
		if err == nil {
			return resp.Node.Value, nil
		}

		_err := err.(etcd.Error)
		if _err.Code != etcd.ErrorCodeKeyNotFound {
			return "", err
		}
		time.Sleep(time.Second * 5)
	}
}

func (m *Manager) GetConfig() (string, error) {
	key := fmt.Sprintf("%s/conf/%s", bts.settings.ETCD.RootPath, m.Name)
	//bts.logger.Debug(key)
	return m.getFromETCD(key)
}

func (m *Manager) monitor() {
	bts.logger.Notice("Start <monitor:%s>", m.Name)
	for {
		for _, bot := range bts.worker.bots {
			alive := bot.Alive()
			//bts.logger.Debug("pid:%d alive:%t", bot.cmd.Process.Pid, alive)
			if !alive {
				bts.worker.DelBot(bot)
			}
		}
		time.Sleep(time.Second * 5)
	}
}

func (m *Manager) StopAllBot() {
	for _, bot := range bts.worker.bots {
		bts.worker.StopBot(bot)
	}
}
