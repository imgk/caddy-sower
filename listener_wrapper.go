package sower

import (
	"net"
	"net/netip"

	"github.com/caddyserver/caddy/v2"

	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(ListenerWrapper{})
}

// ListenerWrapper is ...
type ListenerWrapper struct {
	ExcludeDomain []string `json:"exclude_domains,omitempty"`

	AllowedIPs []string `json:"allowed_ips,omitempty"`
	allowedIPs AddressRange

	lg *zap.Logger
}

func (ListenerWrapper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.sower",
		New: func() caddy.Module { return new(ListenerWrapper) },
	}
}

func (lw *ListenerWrapper) Provision(ctx caddy.Context) error {
	lw.lg = ctx.Logger(lw)

	lw.allowedIPs = make([]netip.Prefix, len(lw.AllowedIPs))
	for k, v := range lw.AllowedIPs {
		p, err := netip.ParsePrefix(v)
		if err != nil {
			return err
		}
		lw.allowedIPs[k] = p
	}

	return nil
}

func (lw *ListenerWrapper) WrapListener(l net.Listener) net.Listener {
	ln := &Listener{
		dl:     lw.ExcludeDomain,
		ar:     lw.allowedIPs,
		lg:     lw.lg,
		ln:     l,
		connCh: make(chan Result),
	}

	go ln.loop()

	return ln
}

var _ caddy.ListenerWrapper = (*ListenerWrapper)(nil)
