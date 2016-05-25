## Announce Interval Variation Middleware

This package provides the announce middleware `varinterval` which randomizes the announce interval.

### Functionality

This middleware will choose random announces and modify the `interval` and `min_interval` fields.
A random number of seconds will be added to the `interval` field and, if desired, also to the `min_interval` field.

Note that if a response is picked for modification and `min_interval` should be changed as well, both `interval` and `min_interval` will be modified by the same amount.

### Use Case

Use this middleware to avoid recurring load spikes on the tracker.
By randomizing the announce interval, load spikes will flatten out after a few cycles.

### Configuration

This middleware provides the following parameters for configuration:

- `modify_response_probability` (float, >0, <= 1) indicates the probability by which a response will be chosen to have its announce intervals modified.
- `max_increase_delta` (int, >0) sets an upper boundary (inclusive) for the amount of seconds added.
- `modify_min_interval` (boolean) whether to modify the `min_interval` field as well.

An example config might look like this:

    chihaya:
      tracker:
        announce_middleware:
          - name: varinterval
            config:
              modify_response_probability: 0.2
              max_increase_delta: 60
              modify_min_interval: true
