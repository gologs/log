[![GoDoc] (https://godoc.org/github.com/gologs/log?status.png)](https://godoc.org/github.com/gologs/log)
[![Circle CI](https://circleci.com/gh/gologs/log.svg?style=svg)](https://circleci.com/gh/gologs/log)
[![Coverage Status](https://coveralls.io/repos/github/gologs/log/badge.svg?branch=master)](https://coveralls.io/github/gologs/log?branch=master)

## Status

This project is currently undergoing active development.
It likely has sharp edges.
Handle (vendor) with care to avoid cuts.

## About

I had some ideas for a logging API that was minimally opinionated.
That should be easy to swap out if needed.
That should be relatively painless to use when building library code intended to be shared across projects.
One that doesn't necessarily dictate the underlying mechanics of the log subsystem.
Something composable that lets you use only the parts your application or library wants.
That lets you add and extend functionality in ways that you need without being over-prescriptive.

This project will not be a kitchen sink of logging utilities.
