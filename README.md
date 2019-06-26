# Gooff [Go(lang) Offline]

Cache all your internet requests when developing

While travelling on and off planes I wanted to play around with external APIs to find the data I was looking for (specifically github's API) and this meant I could hit all the end points I needed while in the airport and continue developing on the plane.

Another great benefit of this library is that it can enable you to have consistent demos when interacting with third party APIs. Try once, demo consistently.

## When's Good To Use

- Having consistent demos when hitting an external API
- Going on and offline periodically for example when travelling on a plane
- Saving internet costs when on mobile
- Demoing in an area with no internet
- API takes a long time to return
- Decrease request count to external API
- When exploring a third party API

## When Not To Use

- To provide use-cases for unit tests. Just no.
- In production, I imagine...

## Install

```bash
go get github.com/Rambatino/gooff
```

## Usage

Only support the side-effects use case for quick switching on and off

Add this to your `main.go`

```go
import _ "github.com/Rambatino/gooff"
```

And that's it!

## Some notes on the package

It will cache all your (http.Status == 200) requests and use the cached value when making repeated requests.

It supports all requests GET, POST, PUT...

As this is only for personal use, initialising in init() using package side effects seems appropriate.
