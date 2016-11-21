package main

import (
	"os/exec"
	"syscall"
)

type Bot struct {
	bid    string
	cmdStr string
	cmd    *exec.Cmd
}

func NewBot(bid string, cmdStr string, cmd *exec.Cmd) *Bot {
	return &Bot{bid, cmdStr, cmd}
}

func (b *Bot) Stop() {
	err := b.cmd.Process.Kill()
	if err != nil {
		bts.logger.Error("stop bot failed:%s", err)
	}
}

//TODO
func (b *Bot) Alive() bool {
	pid := b.cmd.Process.Pid
	//bts.logger.Debug("bot.Alive.pid:%d", pid)
	if err := syscall.Kill(pid, 0); err == nil {
		return true
	}
	return false
}
