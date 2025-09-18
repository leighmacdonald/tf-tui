package console

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
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
	secret          int
	remoteAddress   string
	remotePassword  string
	externalAddress string
	listenAddress   string
}

type RemoteOpts struct {
	ListenAddress string
}

func NewRemote(opts RemoteOpts) (*Remote, error) {
	// TODO better validations
	if opts.ListenAddress == "" {
		return nil, ErrConfig
	}

	return &Remote{listenAddress: opts.ListenAddress}, nil
}

func (l *Remote) Close(_ context.Context) error {
	var err error
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

func (l *Remote) Open() error {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", l.listenAddress)
	if errResolveUDP != nil {
		return errors.Join(errResolveUDP, ErrSetup)
	}

	connection, errListenUDP := net.ListenUDP("udp4", udpAddr)
	if errListenUDP != nil {
		return errors.Join(errListenUDP, ErrSetup)
	}

	l.conn = connection
	l.udpAddr = udpAddr

	return nil
}

// Start initiates the udp network log read loop. DNS names are used/to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (l *Remote) Start(ctx context.Context, receiver Receiver) {
	var (
		insecureCount       = uint64(0)
		serverMessageCounts = map[int]int{}
		logTicker           = time.NewTicker(time.Second * 5)
	)

	slog.Info("Starting log reader", slog.String("listen_addr", l.udpAddr.String()+"/udp"))

	for {
		select {
		case <-logTicker.C:
			var args []any
			for logSecret, count := range serverMessageCounts {
				args = append(args, slog.String("server_id:count", fmt.Sprintf("%d:%d", logSecret, count)))
			}
			slog.Info("Log message counts", args...)
		case <-ctx.Done():
			return
		default:
			buffer := make([]byte, 1024)
			readLen, _, errReadUDP := l.conn.ReadFromUDP(buffer)
			if errReadUDP != nil {
				if errors.Is(errReadUDP, net.ErrClosed) {
					return
				}
				slog.Warn("UDP log read error", slog.String("error", errReadUDP.Error()))

				continue
			}

			var reqSecret int

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

				receiver.Send(0, strings.TrimSpace(string(buffer)))
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

				receiver.Send(int(secret), strings.TrimSpace(line[idx:readLen]))
				reqSecret = int(secret)
			}

			if _, ok := serverMessageCounts[reqSecret]; !ok {
				serverMessageCounts[reqSecret] = 1
			} else {
				serverMessageCounts[reqSecret]++
			}
		}
	}
}
