# ./internal/urlx

This package contains algorithms to operate on URLs.

## ResolveReference

This function has the following signature:

```Go
func ResolveReference(baseURL, path, rawQuery string) (string, error)
```

It solves the problem of computing a composed URL starting from a base URL, an
extra path and a possibly empty raw query. The algorithm will ignore the path and
the query of the base URL and only use the scheme and the host.

For example, assuming the following:

```Go
baseURL := "https://api.ooni.io/antani?foo=bar"
path := "/api/v1/check-in"
query := "bar=baz"
```

This function will return this URL:

```Go
"https://api.ooni.io/ap1/v1/check-in?bar=baz"
```

We need this functionality when implementing communication with the probe services,
where we have a base URL and specific path and optional query for each API.
