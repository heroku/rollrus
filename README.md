# Usage

```go
logrus.AddHook(
  &rollrus.Hook{
    Client: ...A Rollbar client...,
  },
)
```

Note: Uses github.com/heroku/rollbar, not github.com/stvp/rollbar
