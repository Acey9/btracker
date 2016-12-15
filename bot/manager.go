package main

import (
	"bytes"
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
			go bts.worker.StartBot(cmd)
		}
	}

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

	if args[0] == "msync" {
		m.ManualSync()
		return
	}

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
	if resp.Node.Value == resp.PrevNode.Value {
		return
	}
	bts.logger.Notice("config changed.")

	dels := Difference(strings.Split(resp.PrevNode.Value, ";"), strings.Split(resp.Node.Value, ";"))
	bts.logger.Info("delete cnc:%d", len(dels))

	for _, cmdStr := range dels {
		if cmdStr == "" {
			continue
		}
		go bts.worker.StopBot(cmdStr)
	}

	adds := Difference(strings.Split(resp.Node.Value, ";"), strings.Split(resp.PrevNode.Value, ";"))
	bts.logger.Info("new cnc:%d", len(adds))
	for _, cmdStr := range adds {
		if cmdStr == "" {
			continue
		}
		go bts.worker.StartBot(cmdStr)
	}

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

func (m *Manager) ManualSync() {
	values, _ := m.GetConfig()
	for _, cmdStr := range strings.Split(values, ";") {
		go bts.worker.CheckBot(cmdStr)
	}
}

func (m *Manager) StopAllBot() {
	for _, bot := range bts.worker.bots {
		bts.worker.StopBot(bot.cmdStr)
	}
}
