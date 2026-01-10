package tlsutil

import (
	"context"
	"net"
	"net/http"

	utls "github.com/refraction-networking/utls"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
)

// GetClientHelloID 根据平台类型获取对应的 uTLS ClientHelloID
func GetClientHelloID(platformType waCompanionReg.DeviceProps_PlatformType) utls.ClientHelloID {
	switch platformType {
	case waCompanionReg.DeviceProps_CHROME:
		return utls.HelloChrome_120
	case waCompanionReg.DeviceProps_FIREFOX:
		return utls.HelloFirefox_120
	case waCompanionReg.DeviceProps_SAFARI:
		return utls.HelloSafari_16_0
	case waCompanionReg.DeviceProps_EDGE:
		return utls.HelloChrome_120 // Edge 基于 Chromium，使用 Chrome 指纹即可
	case waCompanionReg.DeviceProps_OPERA:
		return utls.HelloChrome_120 // Opera 基于 Chromium，使用 Chrome 指纹即可
	default:
		// 默认返回 Chrome
		return utls.HelloChrome_120
	}
}

// NewUTLSRoundTripper 创建一个支持 uTLS 的 http.RoundTripper
func NewUTLSRoundTripper(clientHelloID utls.ClientHelloID, baseTransport *http.Transport) http.RoundTripper {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport.(*http.Transport).Clone()
	}

	return &utlsRoundTripper{
		base:          baseTransport,
		clientHelloID: clientHelloID,
	}
}

type utlsRoundTripper struct {
	base          *http.Transport
	clientHelloID utls.ClientHelloID
}

func (u *utlsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// 克隆 Transport 以避免并发修改 DialTLSContext
	transport := u.base.Clone()
	transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		host := req.URL.Hostname()
		uConn := utls.UClient(conn, &utls.Config{
			ServerName: host,
		}, u.clientHelloID)

		if err := uConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}

		return uConn, nil
	}

	return transport.RoundTrip(req)
}
