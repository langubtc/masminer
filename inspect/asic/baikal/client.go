package baikal

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/ka2n/masminer/inspect"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	ssh *ssh.Client

	mu         sync.RWMutex
	systemInfo *SystemInfo
}

func (c *Client) MineStop(ctx context.Context) error {
	return runRemoteShell(c.ssh, minerStopCMD)
}

func (c *Client) MineStart(ctx context.Context) error {
	return runRemoteShell(c.ssh, minerStartCMD)
}

func (c *Client) Restart(ctx context.Context) error {
	_, err := outputMinerRPC(c.ssh, "restart", "")
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Reboot(ctx context.Context) error {
	return runRemoteShell(c.ssh, "shutdown -r +5")
}

func (c *Client) SetSSH(client *ssh.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ssh = client
}

func (c *Client) Close() error {
	return c.ssh.Close()
}

func (c *Client) RigInfo(ctx context.Context) (inspect.RigInfo, error) {
	var info inspect.RigInfo
	si, err := c.GetSystemInfo()
	if err != nil {
		return info, err
	}

	info.Rig.IPAddr = c.ssh.LocalAddr().String()
	info.Rig.Hostname = si.Hostname
	info.Rig.Name = inspect.ShortName(si.MACAddr)
	info.MinerType = si.ProductType
	info.Manufacture = manufactureName
	info.HardwareVersion = si.ProductVersion
	info.FirmwareVersion = si.FileSystemVersion
	info.MinerVersion = si.MinerVersion
	info.Algos = Algos(si.ProductType)

	return info, nil
}

func (c *Client) RigStat(ctx context.Context) (inspect.RigStat, error) {
	var stat inspect.RigStat

	ms, err := c.GetStats()
	if err != nil {
		return stat, err
	}

	stat.MHS5s = fmt.Sprintf("%.4f", ms.Summary.MHS5S)
	stat.MHSAvarage = fmt.Sprintf("%.4f", ms.Summary.MHSAv)
	stat.KHS5s = fmt.Sprintf("%.4f", ms.Summary.KHS5S)
	stat.KHSAvarage = fmt.Sprintf("%.4f", ms.Summary.KHSAv)

	stat.Accepted = strconv.Itoa(ms.Summary.Accepted)
	stat.Rejected = strconv.Itoa(ms.Summary.Rejected)
	stat.HardwareErrors = strconv.Itoa(ms.Summary.HardwareErrors)
	stat.Utility = fmt.Sprintf("%.4f", ms.Summary.Utility)

	if len(ms.Stats) != len(ms.Devs) {
		return stat, fmt.Errorf("invalid stats/devs")
	}

	stat.Devices = make([]inspect.DeviceStat, len(ms.Devs))
	for i, ds := range ms.Devs {
		st := ms.Stats[i]

		var dev inspect.DeviceStat
		dev.Chips = st.ChipCount
		dev.Frequency = strconv.Itoa(st.Clock)
		dev.TempChip = fmt.Sprintf("%.1f", ds.Temperature)
		dev.HardwareErrors = strconv.Itoa(ds.HardwareErrors)
		dev.Hashrate = fmt.Sprintf("%.4f", ds.MHS5S)
		stat.Devices[i] = dev
	}

	stat.Pools = make([]inspect.PoolStat, len(ms.Pools))
	for i, pl := range ms.Pools {
		var p inspect.PoolStat
		p.URL = pl.URL
		p.User = pl.User
		p.Algo = pl.Algorithm
		p.Status = pl.Status
		p.StratumActive = pl.StratumActive
		p.Priority = pl.Priority
		p.Getworks = strconv.Itoa(pl.Getworks)
		p.Accepted = strconv.Itoa(pl.Accepted)
		p.Rejected = strconv.Itoa(pl.Rejected)
		p.Discarded = strconv.Itoa(pl.Discarded)
		p.Stale = strconv.Itoa(pl.Stale)
		p.DifficultyAccepted = fmt.Sprintf("%.4f", pl.DifficultyAccepted)
		p.DifficultyRejected = fmt.Sprintf("%.4f", pl.DifficultyRejected)
		p.DifficultyStale = fmt.Sprintf("%.4f", pl.DifficultyStale)
		p.LastShareDifficulty = fmt.Sprintf("%.4f", pl.LastShareDifficulty)
		p.LastShareTime = strconv.Itoa(pl.LastShareTime)
		stat.Pools[i] = p
	}

	return stat, nil
}

func (c *Client) MinerSetting(ctx context.Context) (inspect.MinerSetting, error) {
	var s inspect.MinerSetting

	ms, err := getMinerSetting(c.ssh)
	if err != nil {
		return s, err
	}

	ps, err := getMinerPools(c.ssh)
	if err != nil {
		return s, err
	}

	return getCommonMinerSetting(ms, ps), nil
}

func (c *Client) SetMinerSetting(ctx context.Context, setting inspect.MinerSetting) error {
	var ms MinerSetting
	var ps []PoolSetting

	if err := loadCommonMinerSetting(setting, &ms, &ps); err != nil {
		return err
	}

	return writeMinerAndPoolSetting(c.ssh, ms, ps)
}
