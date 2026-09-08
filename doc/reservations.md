# Reservations

Limitador exposes `Reserve`/`Commit` gRPC RPCs to support token rate limit reservations: estimated
capacity can be held ahead of time and later resolved with the caller's real usage.

The `spec.reservations` field on the `Limitador` CR lets you configure this behavior without having
to hand-manage the operand's container args.

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  reservations:
    enabled: true
    maxFraction: "0.5"
    maxTtl: 60s
```

## `enabled`

Controls whether the `Reserve`/`Commit` gRPC RPCs are available. Defaults to `true`. When set to
`false`, the operator passes `--disable-reservations` to the Limitador process, and calling
`Reserve`/`Commit` returns `UNIMPLEMENTED`.

## `maxFraction`

Maximum fraction of a limit's `max_value` that a single `Reserve` call may hold. Must be greater
than `0` and at most `1`. When set, the operator passes it through as
`--max-reservation-fraction`. Limitador's own default, used when unset, is `0.5`.

## `maxTtl`

Maximum TTL a `Reserve` call may request (e.g. `60s`, `2m`). When set, the operator passes it
through as `--max-reservation-ttl`, in seconds. Limitador's own default, used when unset, is `60s`.
