package combine

import (
	"context"
	"encoding/json"
	"net/netip"
	"reflect"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestEmpty(t *testing.T) {
	input := `combine { }`

	d := caddyfile.NewTestDispenser(input)

	r := CombinedIPRange{}
	err := r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Errorf("unmarshal error: %v", err)
	}

	if len(r.PartsRaw) != 0 {
		t.Errorf("incorrect number of parts parsed")
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err = r.Provision(ctx)
	if err != nil {
		t.Errorf("error provisioning empty 'combine': %v", err)
	}

	ips := r.GetIPRanges(nil)

	if len(ips) != 0 {
		t.Errorf("conjured IPs out of thin air: %v", ips)
	}
}

func TestUnmarshal(t *testing.T) {
	// This can be expressed as a single "static" with two ranges,
	// but static ranges are the only ones that don't require an extra import.
	// Also, we want at least two sub-directives to make things interesting.
	input := `combine {
			  	static 1.1.1.1/24
				static 2001:db8::/32
			  }`

	d := caddyfile.NewTestDispenser(input)

	r := CombinedIPRange{}
	err := r.UnmarshalCaddyfile(d)
	if err != nil {
		t.Errorf("unmarshal error: %v", err)
	}

	if len(r.PartsRaw) != 2 {
		t.Errorf("incorrect number of parts parsed")
	}

	var (
		part0 caddyhttp.StaticIPRange
		part1 caddyhttp.StaticIPRange
	)

	// Check first part.
	err = json.Unmarshal([]byte(r.PartsRaw[0]), &part0)
	if err != nil {
		t.Errorf("error unmarshalling first part: %v", err)
	}
	expected := caddyhttp.StaticIPRange{Ranges: []string{("1.1.1.1/24")}}
	if !reflect.DeepEqual(expected, part0) {
		t.Errorf("first part does not match expectation: expected %v, got %v", expected, part0)
	}

	// Check second part.
	err = json.Unmarshal([]byte(r.PartsRaw[1]), &part1)
	if err != nil {
		t.Errorf("error unmarshalling second part: %v", err)
	}
	expected = caddyhttp.StaticIPRange{Ranges: []string{("2001:db8::/32")}}
	if !reflect.DeepEqual(expected, part1) {
		t.Errorf("second part does not match expectation: expected %v, got %v", expected, part1)
	}
}

func TestProvisionAndQuery(t *testing.T) {
	r := CombinedIPRange{
		PartsRaw: []json.RawMessage{
			mustMarshal(caddyhttp.StaticIPRange{Ranges: []string{"1.1.1.1/24"}}),
			mustMarshal(caddyhttp.StaticIPRange{Ranges: []string{"2001:db8::/32"}}),
		},
	}

	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()

	err := r.Provision(ctx)
	if err != nil {
		t.Errorf("provisioning error: %v", err)
	}

	ips := r.GetIPRanges(nil)
	expected := []netip.Prefix{netip.MustParsePrefix("1.1.1.1/24"), netip.MustParsePrefix("2001:db8::/32")}
	if !reflect.DeepEqual(expected, ips) {
		t.Errorf("expected %v, got %v", expected, ips)
	}
}

func mustMarshal(v caddy.Module) json.RawMessage {
	return caddyconfig.JSONModuleObject(v, "source", v.CaddyModule().ID.Name(), nil)
}
