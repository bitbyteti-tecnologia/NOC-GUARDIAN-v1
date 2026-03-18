//go:build windows

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
)

type agentService struct {
	cfgPath     string
	diskPath    string
	intervalSec int
	serverURL   string
	tenantID    string
}

func isWindowsService() bool {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return isSvc
}

func runAsWindowsService(cfgPath, diskPath string, intervalSec int, serverURL, tenantID string) {
	s := &agentService{
		cfgPath:     cfgPath,
		diskPath:    diskPath,
		intervalSec: intervalSec,
		serverURL:   serverURL,
		tenantID:    tenantID,
	}

	if err := svc.Run("NOCGuardianAgent", s); err != nil {
		log.Fatalf("service run error: %v", err)
	}
}

func (s *agentService) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const accepts = svc.AcceptStop | svc.AcceptShutdown

	status <- svc.Status{State: svc.StartPending}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_ = runAgent(s.cfgPath, s.diskPath, s.intervalSec, s.serverURL, s.tenantID, stop)
		close(done)
	}()

	status <- svc.Status{State: svc.Running, Accepts: accepts}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				close(stop)
				<-done
				status <- svc.Status{State: svc.Stopped}
				return false, 0
			default:
			}
		case <-done:
			status <- svc.Status{State: svc.Stopped}
			return false, 0
		}
	}
}

