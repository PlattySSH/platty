package sshx

import (
	"testing"
	"time"
)

const statsSample = `cpu  100 0 50 800 50 0 0 0 0 0
cpu0 50 0 25 400 25 0 0 0 0 0
cpu1 50 0 25 400 25 0 0 0 0 0
intr 12345 0 0
ctxt 999
MemTotal:        4000000 kB
MemFree:          500000 kB
MemAvailable:    3000000 kB
SwapTotal:       2000000 kB
SwapFree:        1500000 kB
Buffers:          100000 kB
12345.67 23456.78
0.42 0.30 0.20 1/234 5678
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 999999 10 0 0 0 0 0 0 999999 10 0 0 0 0 0 0
  eth0: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0
@@df
/dev/sda1 10000000 2500000 7500000 25% /
@@os
Linux 6.8.0-45-generic
osiris
PRETTY_NAME="Ubuntu 24.04.1 LTS"
`

func TestParseStats(t *testing.T) {
	r, err := parseStats([]byte(statsSample))

	if err != nil {
		t.Fatal(err)
	}

	s := r.stats

	if s.Cores != 2 || r.cpuTotal != 1000 || r.idle != 850 {
		t.Errorf("cpu: cores=%d total=%d idle=%d", s.Cores, r.cpuTotal, r.idle)
	}

	if s.MemTotal != 4000000*1024 || s.MemUsed != 1000000*1024 {
		t.Errorf("mem: %d/%d", s.MemUsed, s.MemTotal)
	}

	if s.Uptime != 12345*time.Second || s.Load1 != 0.42 {
		t.Errorf("uptime=%v load=%v", s.Uptime, s.Load1)
	}

	if r.rx != 1000 || r.tx != 2000 {
		t.Errorf("network without lo: rx=%d tx=%d", r.rx, r.tx)
	}

	if s.DiskTotal != 10000000*1024 || s.DiskUsed != 2500000*1024 {
		t.Errorf("disk: %d/%d", s.DiskUsed, s.DiskTotal)
	}

	if s.Load5 != 0.30 || s.Load15 != 0.20 || s.SwapUsed != 500000*1024 || s.DiskAvail != 7500000*1024 {
		t.Errorf("load5=%v load15=%v swap=%d avail=%d", s.Load5, s.Load15, s.SwapUsed, s.DiskAvail)
	}

	if s.OS != "Ubuntu 24.04.1 LTS" || s.Kernel != "Linux 6.8.0-45-generic" || s.Hostname != "osiris" {
		t.Errorf("os=%q kernel=%q host=%q", s.OS, s.Kernel, s.Hostname)
	}

	if _, err := parseStats([]byte("Darwin has no /proc\n")); err != ErrNoStats {
		t.Errorf("without /proc: %v", err)
	}
}
