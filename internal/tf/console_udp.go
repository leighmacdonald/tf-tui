package tf

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"
)

type srcdsPacket byte

const (
	// Normal log messages (unsupported).
	s2aLogString srcdsPacket = 0x52
	// Sent when using sv_logsecret.
	s2aLogString2 srcdsPacket = 0x53
)

var (
	ErrResolve    = errors.New("failed to resolve UDP address")
	ErrUDPListen  = errors.New("failed to listen on UDP address")
	ErrUDPSetup   = errors.New("failed to configure logaddress")
	ErrRateLimit  = errors.New("rate limited")
	ErrUnknownIP  = errors.New("unknown source ip")
	ErrSecretAuth = errors.New("failed secret auth")
)

// SRCDSListener handles reading inbound srcds log packets.
type SRCDSListener struct {
	udpAddr         *net.UDPAddr
	conn            *net.UDPConn
	secret          int64
	remoteAddress   string
	remotePassword  string
	externalAddress string
	listenAddress   string
	logBroadcaster  *LogBroadcaster
}

type SRCDSListenerOpts struct {
	ExternalAddress string
	ListenAddress   string
	RemoteAddress   string
	RemotePassword  string
	Secret          int64
}

func NewSRCDSListener(broadcaster *LogBroadcaster, opts SRCDSListenerOpts) (*SRCDSListener, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", opts.ListenAddress)
	if errResolveUDP != nil {
		return nil, errors.Join(errResolveUDP, ErrResolve)
	}

	connection, errListenUDP := net.ListenUDP("udp4", udpAddr)
	if errListenUDP != nil {
		return nil, errors.Join(errListenUDP, ErrUDPListen)
	}

	if opts.ExternalAddress == "" {
		opts.ExternalAddress = opts.ListenAddress
	}

	return &SRCDSListener{
		udpAddr:         udpAddr,
		conn:            connection,
		remoteAddress:   opts.RemoteAddress,
		remotePassword:  opts.RemotePassword,
		secret:          opts.Secret,
		externalAddress: opts.ExternalAddress,
		listenAddress:   opts.ListenAddress,
		logBroadcaster:  broadcaster,
	}, nil
}

func (l *SRCDSListener) setup(ctx context.Context) error {
	conn := newRconConnection(l.remoteAddress, l.remotePassword)
	_, errExec := conn.exec(ctx, "logaddress_add "+l.listenAddress, false)
	if errExec != nil {
		return errExec
	}

	resp, err := conn.exec(ctx, "logaddress_list", false)
	if err != nil {
		return err
	}

	if !strings.Contains(resp, l.externalAddress) {
		return ErrUDPSetup
	}

	return nil
}

// Start initiates the udp network log read loop. DNS names are used/to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (l *SRCDSListener) Start(ctx context.Context) {
	defer func() {
		if errConnClose := l.conn.Close(); errConnClose != nil {
			slog.Error("Failed to close connection cleanly", slog.String("error", errConnClose.Error()))
		}
	}()

	var (
		insecureCount = uint64(0)
		buffer        = make([]byte, 1024)
	)

	slog.Info("Starting log reader", slog.String("listen_addr", l.udpAddr.String()+"/udp"))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			clear(buffer)
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

				l.logBroadcaster.Send(strings.TrimSpace(string(buffer)))
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

				l.logBroadcaster.Send(strings.TrimSpace(line[idx:readLen]))
			}
		}
	}
}
