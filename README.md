# What

Rollrus is what happens when [Logrus](https://github.com/Sirupsen/logrus) meets [Roll](https://github.com/stvp/roll).

When a .Error, .Fatal or .Panic logging function is called, report the details to rollbar via a Logrus hook.

Delivery is synchronous to help ensure that logs are delivered.

# Usage

Examples available in the [tests](https://github.com/heroku/rollrus/blob/master/rollrus_test.go) or on [GoDoc](https://godoc.org/github.com/heroku/rollrus).
