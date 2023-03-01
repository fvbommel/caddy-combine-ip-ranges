# IP prefix combining module for Caddy

This module retrieves IP prefixes from other modules
and combines them into a single list of IP prefixes
for use in Caddy `trusted_proxies` directives.

NOTE: it doesn't actually merge adjacent or overlapping prefixes,
it just puts all of the sub-results into a big list and returns that.

## Example config

An example configuration you might use while experimenting
with different ways to put your site behind Cloudflare:

```Caddy
trusted_proxies combine {
    # For access via Cloudflare directly, using github.com/WeidiDeng/caddy-cloudflare-ip
    cloudflare
    # For access using cloudflared container on the local Docker bridge network, using github.com/fvbommel/caddy-dns-ip-range
    dns cloudflared
}
```

This will trust both public Cloudflare IPs
and the one `cloudflared` is "borrowing" on your internal network
(assuming it's registered in the local DNS).

There are no other settings, though you can of course pass settings to each individual sub-directive:

```Caddy
trusted_proxies combine {
    cloudflare {
        interval 12h
        timeout 15s
    }
    dns cloudflared {
        interval 1m
    }
}
```
