package upnp

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
	"golang.org/x/sync/errgroup"
)

func New(externalPort uint16, internalPort uint16) *UPNPManager {
	return &UPNPManager{
		externalPort: externalPort,
		internalPort: internalPort,
	}
}

type UPNPManager struct {
	externalPort uint16
	internalPort uint16
}

func (u UPNPManager) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 15)

	u.updateMapping(ctx)

	for {
		select {
		case <-ticker.C:
			u.updateMapping(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (u UPNPManager) updateMapping(ctx context.Context) error {
	client, err := findRouterClient(ctx)
	if err != nil {
		return err
	}

	externalIP, err := client.GetExternalIPAddress()
	if err != nil {
		return err
	}
	slog.Info(externalIP)

	return client.AddPortMapping(
		"",
		u.externalPort,
		"UDP",
		// Some routers might not support this being different to the external
		// port number.
		u.internalPort,
		client.LocalAddr().String(),
		true,
		"tf-tui",
		3600,
	)
}

type routerClient interface {
	AddPortMapping(
		NewRemoteHost string,
		NewExternalPort uint16,
		NewProtocol string,
		NewInternalPort uint16,
		NewInternalClient string,
		NewEnabled bool,
		NewPortMappingDescription string,
		NewLeaseDuration uint32,
	) (err error)

	GetExternalIPAddress() (
		NewExternalIPAddress string,
		err error,
	)

	LocalAddr() net.IP
}

func findRouterClient(ctx context.Context) (routerClient, error) {
	tasks, _ := errgroup.WithContext(ctx)
	// Request each type of client in parallel, and return what is found.
	var ip1Clients []*internetgateway2.WANIPConnection1
	tasks.Go(func() error {
		var err error
		ip1Clients, _, err = internetgateway2.NewWANIPConnection1Clients()
		return err
	})
	var ip2Clients []*internetgateway2.WANIPConnection2
	tasks.Go(func() error {
		var err error
		ip2Clients, _, err = internetgateway2.NewWANIPConnection2Clients()
		return err
	})
	var ppp1Clients []*internetgateway2.WANPPPConnection1
	tasks.Go(func() error {
		var err error
		ppp1Clients, _, err = internetgateway2.NewWANPPPConnection1Clients()
		return err
	})

	if err := tasks.Wait(); err != nil {
		return nil, err
	}

	switch {
	case len(ip2Clients) == 1:
		return ip2Clients[0], nil
	case len(ip1Clients) == 1:
		return ip1Clients[0], nil
	case len(ppp1Clients) == 1:
		return ppp1Clients[0], nil
	default:
		return nil, errors.New("multiple or no services found")
	}
}
