package main

import (
	"github.com/BurntSushi/toml"
	"github.com/astaxie/beego/logs"
)

type BotSettings struct {
	Title      string
	Version    string
	Debug      bool
	BotBinPath string
	Log        LogSettings
	ETCD       ETCDSettings
}

func NewSettings(settingsFile string, settings *BotSettings) error {
	_, err := toml.DecodeFile(settingsFile, settings)
	return err
}

var LogLevelMap = map[string]int{
	"DEBUG":  logs.LevelDebug,
	"INFO":   logs.LevelInfo,
	"NOTICE": logs.LevelNotice,
	"WARN":   logs.LevelWarning,
	"ERROR":  logs.LevelError,
}

type ETCDSettings struct {
	Endpoints []string
	Timeout   int
	Username  string
	Password  string
	RootPath  string
	GroupBy   string
}

type LogSettings struct {
	Stdout bool
	Path   string
	Level  string `toml:"Level"`
}

func (ls LogSettings) BeeLevel() int {
	l, ok := LogLevelMap[ls.Level]
	if !ok {
		panic("Config error: invalid log level: " + ls.Level)
	}
	return l
}
