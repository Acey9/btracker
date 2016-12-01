package main

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"time"
)

type SysInfo struct {
	Load *load.AvgStat
	VMem *mem.VirtualMemoryStat
	Swap *mem.SwapMemoryStat
	Disk *disk.UsageStat
	CPU  []float64
}

func NewSysInfo() *SysInfo {
	si := &SysInfo{}
	ld, err := load.Avg()
	if err != nil {
		fmt.Println(err)
	} else {
		si.Load = ld
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		fmt.Println(err)
	} else {
		si.Swap = swap
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println(err)
	} else {
		si.VMem = vm
	}

	diskUsage, err := disk.Usage("/")
	if err != nil {
		fmt.Println(err)
	} else {
		si.Disk = diskUsage
	}

	cpuPer, err := cpu.Percent(time.Duration(0), false)
	if err != nil {
		fmt.Println(err)
	} else {
		si.CPU = cpuPer
	}
	return si
}
