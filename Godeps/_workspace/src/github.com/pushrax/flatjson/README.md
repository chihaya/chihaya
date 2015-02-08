# flatjson [![Build Status](https://api.travis-ci.org/pushrax/flatjson.svg?branch=master)](https://travis-ci.org/pushrax/flatjson)

flatjson is a Go package for collapsing structs into a flat map, which can then be JSON encoded.
The map values are pointers to the original struct fields, so it does not need to be regenerated when the values are updated.

Example use case:

```json
{
  "Connections": {
    "Open": 2,
    "Accepted": 4
  },
  "ResponseTime": {
    "P50": 0.045775,
    "P90": 0.074299,
    "P95": 0.096207
  },
  "Peers.IPv6": {
    "Current": 0,
    "Joined": 0,
    "Left": 0,
    "Reaped": 0,
    "Completed": 0,
    "Seeds": {
      "Current": 0,
      "Joined": 0,
      "Left": 0,
      "Reaped": 0
    }
  },
  "Memory": {
    "Alloc": 682208,
    "TotalAlloc": 1032488,
    "Sys": 5441784,
    "Lookups": 28,
    "Mallocs": 3326,
    "Frees": 2567
  }
}
```

is instead serialized as:

```json
{
  "Connections.Accepted": 4,
  "Connections.Open": 2,
  "Memory.Alloc": 682208,
  "Memory.Frees": 2567,
  "Memory.Lookups": 281,
  "Memory.Mallocs": 3326,
  "Memory.Sys": 5441784,
  "Memory.TotalAlloc": 1032488,
  "Peers.IPv6.Completed": 0,
  "Peers.IPv6.Current": 0,
  "Peers.IPv6.Joined": 0,
  "Peers.IPv6.Left": 0,
  "Peers.IPv6.Reaped": 0,
  "Peers.IPv6.Seeds.Current": 0,
  "Peers.IPv6.Seeds.Joined": 0,
  "Peers.IPv6.Seeds.Left": 0,
  "Peers.IPv6.Seeds.Reaped": 0,
  "ResponseTime.P50": 0.045775,
  "ResponseTime.P90": 0.074299,
  "ResponseTime.P95": 0.096207
}
```
