package network

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/leighmacdonald/tf-tui/internal/encoding"
)

// httpClientV4 creates a http client only capable of speaking ipv4. This is used when querying the external
// ip so it returns a usable ip. It must use the v4 stack as thats all that srcds supports.
func httpClientV4() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _ string, addr string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "tcp4", addr)
			},
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 6 * time.Second,
		},
	}

	return client
}

type IPInfo struct {
	IP  string `json:"ip"`
	ISP struct {
		ASN string `json:"asn"`
		Org string `json:"org"`
		ISP string `json:"isp"`
	} `json:"isp"`
	Location struct {
		Country     string  `json:"country"`
		CountryCode string  `json:"country_code"`
		City        string  `json:"city"`
		State       string  `json:"state"`
		Zipcode     string  `json:"zipcode"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		Timezone    string  `json:"timezone"`
		Localtime   string  `json:"localtime"`
	} `json:"location"`
	Risk struct {
		IsMobile     bool `json:"is_mobile"`
		IsVPN        bool `json:"is_vpn"`
		IsTor        bool `json:"is_tor"`
		IsProxy      bool `json:"is_proxy"`
		IsDatacenter bool `json:"is_datacenter"`
		RiskScore    int  `json:"risk_score"`
	} `json:"risk"`
}

var ErrQueryIP = errors.New("failed to query ip")

// FetchIPInfo queries a remote api and returns the public routable ip of the client.
func FetchIPInfo(ctx context.Context) (*IPInfo, error) {
	return FetchJSON[IPInfo](ctx, "https://api.ipquery.io/?format=json")
}

// FetchJSON will query a json http service using a generic type for receiving results.
func FetchJSON[T any](ctx context.Context, url string) (*T, error) {
	client := httpClientV4()

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errReq != nil {
		return nil, errors.Join(errReq, ErrQueryIP)
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Join(errResp, ErrQueryIP)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", slog.String("error", err.Error()))
		}
	}()

	info, errInfo := encoding.UnmarshalJSON[T](resp.Body)
	if errInfo != nil {
		return nil, errors.Join(errInfo, ErrQueryIP)
	}

	return &info, nil
}
