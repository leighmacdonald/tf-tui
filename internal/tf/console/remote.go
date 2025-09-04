package console

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"github.com/leighmacdonald/tf-tui/internal/tf/rcon"
)

type srcdsPacket byte

const (
	// Normal log messages.
	s2aLogString srcdsPacket = 0x52
	// Sent when using sv_logsecret for "authentication".
	s2aLogString2 srcdsPacket = 0x53
)

// Remote handles reading inbound srcds log packets.
type Remote struct {
	udpAddr         *net.UDPAddr
	conn            *net.UDPConn
	secret          int64
	remoteAddress   string
	remotePassword  string
	externalAddress string
	listenAddress   string
}

type SRCDSListenerOpts struct {
	ExternalAddress string
	ListenAddress   string
	RemoteAddress   string
	RemotePassword  string
	Secret          int64
}

func NewRemote(opts SRCDSListenerOpts) (*Remote, error) {
	if opts.ExternalAddress == "" {
		opts.ExternalAddress = opts.ListenAddress
	}

	// TODO better validations
	if opts.RemoteAddress == "" {
		return nil, ErrConfig
	}

	return &Remote{
		remoteAddress:   opts.RemoteAddress,
		remotePassword:  opts.RemotePassword,
		secret:          opts.Secret,
		externalAddress: opts.ExternalAddress,
		listenAddress:   opts.ListenAddress,
	}, nil
}

func (l *Remote) Close(ctx context.Context) error {
	var err error
	// Be cool and remove ourselves from the log address list.
	conn := rcon.New(l.remoteAddress, l.remotePassword)
	_, errExec := conn.Exec(ctx, "logaddress_del "+l.externalAddress, false)
	if errExec != nil {
		err = errors.Join(err, errExec)
	}

	if l.conn != nil {
		if errConnClose := l.conn.Close(); errConnClose != nil {
			err = errors.Join(err, errConnClose)
		}
	}

	if err != nil {
		return errors.Join(err, ErrClose)
	}

	return nil
}

func (l *Remote) Open(ctx context.Context) error {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", l.listenAddress)
	if errResolveUDP != nil {
		return errors.Join(errResolveUDP, ErrSetup)
	}

	connection, errListenUDP := net.ListenUDP("udp4", udpAddr)
	if errListenUDP != nil {
		return errors.Join(errListenUDP, ErrSetup)
	}

	l.conn = connection

	conn := rcon.New(l.remoteAddress, l.remotePassword)
	_, errExec := conn.Exec(ctx, "logaddress_add "+l.externalAddress, false)
	if errExec != nil {
		return errors.Join(errExec, ErrOpen)
	}

	resp, err := conn.Exec(ctx, "logaddress_list", false)
	if err != nil {
		return errors.Join(err, ErrOpen)
	}

	if !strings.Contains(resp, l.externalAddress) {
		return ErrOpen
	}

	return nil
}

// Start initiates the udp network log read loop. DNS names are used/to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (l *Remote) Start(ctx context.Context, receiver Receiver) {
	insecureCount := uint64(0)

	slog.Info("Starting log reader", slog.String("listen_addr", l.udpAddr.String()+"/udp"))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			buffer := make([]byte, 1024)
			readLen, _, errReadUDP := l.conn.ReadFromUDP(buffer)
			if errReadUDP != nil {
				slog.Warn("UDP log read error", slog.String("error", errReadUDP.Error()))

				continue
			}

			switch srcdsPacket(buffer[4]) {
			case s2aLogString: // Legacy/insecure format (no secret)
				// Only care if we actually set a secret
				if l.secret > 0 {
					if insecureCount%100 == 0 {
						slog.Error("Using unsupported log packet type 0x52",
							slog.Int64("count", int64(insecureCount+1))) //nolint:gosec
					}
					insecureCount++
				}

				receiver.Send(strings.TrimSpace(string(buffer)))
			case s2aLogString2: // Secure format (with secret)
				line := string(buffer)

				idx := strings.Index(line, "L ")
				if idx == -1 {
					slog.Warn("Received malformed log message: Failed to find marker")

					continue
				}

				secret, errConv := strconv.ParseInt(line[5:idx], 10, 32)
				if errConv != nil {
					slog.Error("Received malformed log message: Failed to parse secret",
						slog.String("error", errConv.Error()))

					continue
				}

				if secret > 0 && secret != l.secret {
					slog.Warn("Received unauthenticated log message: Invalid secret")

					continue
				}

				receiver.Send(strings.TrimSpace(line[idx:readLen]))
			}
		}
	}
}
