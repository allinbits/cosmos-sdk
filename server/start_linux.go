package server

import (
	"fmt"
	"os"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/tendermint/tendermint/node"
)

// On Linux, we implement sd_notify protocol:
// https://www.freedesktop.org/software/systemd/man/sd_notify.html#Description

func init() {
	notifyReady = func() {
		if ok, err := daemon.SdNotify(false, daemon.SdNotifyReady); !ok && err != nil {
			fmt.Fprintf(os.Stderr, "couldn't notify systemd: %v", err)
		}
	}

	sleepLoop = func(tmNode *node.Node) {
		interval, err := daemon.SdWatchdogEnabled(false)
		if err != nil && interval == 0 {
			fmt.Fprintf(os.Stderr, "couldn't acquire watchdog information: %v", err)
		}

		if interval != 0 {
			// run forever (the node will not be returned)
			for {
				select {
				case <-time.After(interval / 2):
					if tmNode.IsRunning() {
						if ok, err := daemon.SdNotify(false, daemon.SdNotifyWatchdog); !ok && err != nil {
							fmt.Fprintf(os.Stderr, "couldn't notify systemd: %v", err)
						}
					}
				}
			}
		} else {
			select {}
		}
	}
}
