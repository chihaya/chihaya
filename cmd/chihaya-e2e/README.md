# chihaya-e2e
A very simple tool to black-box test a bittorrent tracker.

This tool uses [github.com/anacrolix/torrent/tracker](github.com/anacrolix/torrent/tracker) to make a UDP and an HTTP announce to given trackers.
It is used by chihaya for end-to-end testing the tracker during CI.
