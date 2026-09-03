# TODO

## Bootstrap a `mmw` development environment.

Provide a command that bootstrap a complete Modular Monolith Workspace with a generated ready to use module.

## Choose between the OVYA SQL querier (or other solution) and GraphQL

## Add middleware

```txt
RateLimit   // Rate limiting
BasicAuth   // Basic authentication
Timeout(5*time.Second)
Compress    // Gzip compression
Secure      // Security headers
BodyLimit(1*MB)
ETag             // ETag caching
NoCache          // Cache prevention
Static("./public)" // Static file serving
```

## Enforce Git Conventional Commits

- https://github.com/commitizen/cz-cli (not maitained
- https://github.com/lintingzhen/commitizen-go (go
- https://github.com/conventional-changelog/commitlint
