package combine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(CombinedIPRange{})
}

// This module combines the prefixes returned by several other IP source plugins.
// In a caddyfile, you can specify these in the block following the "combine" tag.
type CombinedIPRange struct {
	// The IP ranges to combine.
	PartsRaw []json.RawMessage `json:"parts,omitempty" caddy:"namespace=http.ip_sources inline_key=source"`

	// The provisioned IP range sources to combine.
	parts []caddyhttp.IPRangeSource

	// Canceled when the module is being cleaned up.
	ctx caddy.Context

	// The logger.
	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (c CombinedIPRange) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.ip_sources.combine",
		New: func() caddy.Module { return new(CombinedIPRange) },
	}
}

func (c *CombinedIPRange) Provision(ctx caddy.Context) error {
	// Initialize internal fields.
	c.ctx = ctx
	c.logger = ctx.Logger()

	// Sanity check.
	if len(c.PartsRaw) == 0 {
		c.logger.Warn("combine ip range: no sub-ranges provided")
	}

	// Provision parts.
	rawParts, err := ctx.LoadModule(c, "PartsRaw")
	if err != nil {
		return fmt.Errorf("loading sub-range modules: %v", err)
	}
	for _, val := range rawParts.([]any) {
		part, ok := val.(caddyhttp.IPRangeSource)
		if !ok {
			return fmt.Errorf("%T is not an IP range source", part)
		}
		c.parts = append(c.parts, part)
	}

	return nil
}

func (c *CombinedIPRange) GetIPRanges(r *http.Request) (result []netip.Prefix) {
	for _, part := range c.parts {
		result = append(result, part.GetIPRanges(r)...)
	}

	// I suppose we could be fancy here and merge overlapping or adjacent ranges
	// where possible, but there's not much point.

	return result
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
//
// An example configuration you might use while
// experimenting with different ways to put your site behind Cloudflare:
//
//	combine {
//	   # For access via Cloudflare directly, using github.com/WeidiDeng/caddy-cloudflare-ip
//	   cloudflare
//	   # For access using cloudflared container on the same Docker network, using github.com/fvbommel/caddy-dns-ip-range
//	   dns cloudflared
//	}
func (c *CombinedIPRange) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	if !d.Next() || d.Val() != "combine" {
		d.Err("expected 'combine'")
	}

	if d.NextArg() {
		return d.Err("unexpected argument")
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		modID := "http.ip_sources." + d.Val()
		mod, err := caddyfile.UnmarshalModule(d, modID)
		if err != nil {
			return err
		}
		source, ok := mod.(caddyhttp.IPRangeSource)
		if !ok {
			return fmt.Errorf("module %s (%T) is not an IP range source", modID, mod)
		}
		jsonSource := caddyconfig.JSONModuleObject(
			source,
			"source",
			source.(caddy.Module).CaddyModule().ID.Name(),
			nil,
		)
		c.PartsRaw = append(c.PartsRaw, jsonSource)
	}

	return nil
}

// Interface guards
var (
	_ caddy.Module            = (*CombinedIPRange)(nil)
	_ caddy.Provisioner       = (*CombinedIPRange)(nil)
	_ caddyfile.Unmarshaler   = (*CombinedIPRange)(nil)
	_ caddyhttp.IPRangeSource = (*CombinedIPRange)(nil)
)
