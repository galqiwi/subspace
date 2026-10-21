# subspace

Expose a local HTTP service through a public relay server, without touching firewall or NAT rules.

The machine with the local service dials out to the server; requesters reach it by passing its id as a bearer token.

```
requester --HTTP--> subspace server (public) <--websocket-- subspace client --> localhost:<port>
```

### Install

```
uv tool install subspace-tunnel
```

or run without installing: `uvx subspace-tunnel ...`

### Server

```yaml
# server.yaml
listen: ":8080"
client_tokens:
  - "some-long-random-token"
```

```
subspace server --config server.yaml
```

TLS is out of scope, put the server behind Caddy/nginx/etc.

### Client

```
subspace client --server https://relay.example.com --token some-long-random-token --id some-long-random-id --port 8000
```

`--upstream-authorization "Bearer <token>"` sets the `Authorization` header sent to the local port. Without it, the header is stripped.

Only one client can hold an id at a time. The id is freed when the client disconnects, or ~20s after it dies.

### Requester

```
curl https://relay.example.com/ -H "Authorization: Bearer some-long-random-id"
```

The id is effectively an api key: anyone who knows it can use the exposed service.
