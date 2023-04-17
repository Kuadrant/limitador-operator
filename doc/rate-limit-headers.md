# Rate Limit Headers

It enables RateLimit Header Fields for HTTP as specified in
[Rate Limit Headers Draft](https://datatracker.ietf.org/doc/id/draft-polli-ratelimit-headers-03.html)

```yaml
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-sample
spec:
  rateLimitHeaders: DRAFT_VERSION_03
```

Current valid values are:
* *DRAFT_VERSION_03* (ref: https://datatracker.ietf.org/doc/id/draft-polli-ratelimit-headers-03.html)
* *NONE*

By default, when `spec.rateLimitHeaders` is *null*, `--rate-limit-headers` command line arg is not
included in the limitador's deployment.
