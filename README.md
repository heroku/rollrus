# Usage

```go
logrus.AddHook(
  &rollrus.Hook{
    client: ...A Rollbar client...,
  },
)
```

Note: Uses github.com/heroku/rollbar, not github.com/stvp/rollbar
