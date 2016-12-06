package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	cgroupfs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/configs"
)

type stat struct {
	ts  time.Time
	cpu cgroups.CpuStats
}

var (
	sysrq          = flag.String("sysrq", "l", "Sysrq command to send")
	triggerPercent = flag.Int("trigger-percent", 10, "CPU availability percentage below which to trigger sysrq")
	period         = flag.Duration("period", 50*time.Millisecond, "CPU usage monitoring period")
	last           *stat
)

const sysrqFile = "/proc/sysrq-trigger"

func main() {
	flag.Parse()
	cgroupManager, err := getCgroupManager()
	if err != nil {
		glog.Fatalf("failed to create cgroupmanager: %v", err)
	}
	numCPU := float64(runtime.NumCPU())
	cpuMonitor := func() {
		usage, err := getCpuUsagePercent(cgroupManager)
		if err != nil {
			glog.Errorf("failed to get CPU usage: %v", err)
			return
		}
		glog.V(1).Infof("%q: CPU Usage: %f", time.Now().String(), usage*100)
		avail := (numCPU - usage) * 100
		if avail <= float64(*triggerPercent) {
			glog.Infof("Triggering sysrq since CPU availability is %f which is lesser than trigger value %d", avail, *triggerPercent)
			triggerSysrq()
		}
	}
	for _ = range time.NewTicker(*period).C {
		cpuMonitor()
	}
	glog.Fatalf("Unexpected code execution")
}

func triggerSysrq() {
	if err := ioutil.WriteFile(sysrqFile, []byte(*sysrq), 0644); err != nil {
		glog.Errorf("failed to trigger sysrq: %v", err)
	}
}

func getCpuUsagePercent(cm *cgroupfs.Manager) (float64, error) {
	cgroupStats, err := cm.GetStats()
	if err != nil {
		return 0, err
	}
	s := &stat{
		ts:  time.Now(),
		cpu: cgroupStats.CpuStats,
	}
	glog.V(2).Infof("Got stats: %+v", cgroupStats.CpuStats)
	instUsage, err := instCpuStats(last, s)
	if err != nil {
		return 0, err
	}
	last = s
	return instUsage, nil
}

func getCgroupManager() (*cgroupfs.Manager, error) {
	paths, err := libcontainer.GetCgroupSubsystems()
	if err != nil {
		return nil, err
	}
	cgroupPaths := makeCgroupPaths(paths.MountPoints, "/")
	return &cgroupfs.Manager{
		Cgroups: &configs.Cgroup{
			Name: "/",
		},
		Paths: cgroupPaths,
	}, nil
}

func makeCgroupPaths(mountPoints map[string]string, name string) map[string]string {
	cgroupPaths := make(map[string]string, len(mountPoints))
	for key, val := range mountPoints {
		cgroupPaths[key] = path.Join(val, name)
	}

	return cgroupPaths
}

func instCpuStats(last, cur *stat) (float64, error) {
	if last == nil {
		return 0, nil
	}
	if !cur.ts.After(last.ts) {
		return 0, fmt.Errorf("container stats move backwards in time")
	}
	if len(last.cpu.CpuUsage.PercpuUsage) != len(cur.cpu.CpuUsage.PercpuUsage) {
		return 0, fmt.Errorf("different number of cpus")
	}
	timeDelta := cur.ts.Sub(last.ts)
	if timeDelta <= time.Millisecond {
		return 0, fmt.Errorf("time delta unexpectedly small")
	}
	// Nanoseconds to gain precision and avoid having zero seconds if the
	// difference between the timestamps is just under a second
	timeDeltaNs := uint64(timeDelta.Nanoseconds())
	convertToRate := func(lastValue, curValue uint64) (float64, error) {
		if curValue < lastValue {
			return 0, fmt.Errorf("cumulative stats decrease")
		}
		valueDelta := curValue - lastValue
		// Use float64 to keep precision
		return float64(valueDelta) / float64(timeDeltaNs), nil
	}
	return convertToRate(last.cpu.CpuUsage.TotalUsage, cur.cpu.CpuUsage.TotalUsage)
}
