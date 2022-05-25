# LBRY Tracker

The LBRY tracker is a server that helps peers find each other. It was forked from [Chihaya](https://github.com/chihaya/chihaya), an open-source [BitTorrent tracker](https://en.wikipedia.org/wiki/BitTorrent_tracker).


## Installation and Usage

### Building from HEAD

In order to compile the project, the [latest stable version of Go] and knowledge of a [working Go environment] are required.

```sh
git clone git@github.com:lbryio/tracker.git
cd tracker
go build ./cmd/chihaya
./chihaya --help
```

[latest stable version of Go]: https://golang.org/dl
[working Go environment]: https://golang.org/doc/code.html

### Testing

The following will run all tests and benchmarks.
Removing `-bench` will just run unit tests.

```sh
go test -bench $(go list ./...)
```

The tracker executable contains a command to end-to-end test a BitTorrent tracker.
See

```sh
tracker --help
```

### Configuration

Configuration of the tracker is done via one YAML configuration file.
The `dist/` directory contains an example configuration file.
Files and directories under `docs/` contain detailed information about configuring middleware, storage implementations, architecture etc.

This is an example for an UDP server running on 6969 with metrics enabled. Remember to **change the private key** to some random string.

```
---
tracker:
  announce_interval: "30m"
  min_announce_interval: "15m"
  metrics_addr: "0.0.0.0:6880"
  udp:
    addr: "0.0.0.0:6969"
    max_clock_skew: "10s"
    private_key: ">>>>CHANGE THIS TO SOME RANDOM THING<<<<"
    enable_request_timing: false
    allow_ip_spoofing: false
    max_numwant: 100
    default_numwant: 50
    max_scrape_infohashes: 50
  storage:
    name: "memory"
    config:
      gc_interval: "3m"
      peer_lifetime: "31m"
      shard_count: 1024
      prometheus_reporting_interval: "1s"
```

# Running from Docker

This section assumes `docker` and `docker-compose` to be installed on a Linux distro. Please check official docs on how to install [Docker Engine](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/).

## Docker Compose from lbry/tracker
In order to define a tracker service and let Docker Compose manage it, create a file named `docker-compose.yml` with:
```
version: "3"
services:
  tracker:
    image: lbry/tracker
    command: --config /config/conf.yml
    volumes:
      - .:/config
    network_mode: host
    restart: always
```
Unfortunately the tracker does not work without `network_mode: host` due some bug with UDP on Docker. In this mode, firewall configuration needs to be done manually. If using `ufw`, try `ufw allow 6969`.

Now, move the configuration to the same directory as `docker-compose.yml`, naming it `conf.yml`. If it is not ready, check the configuration section above.

Start the tracker by running the following in the same directory as the compose file:
`docker-compose up -d`
Logs can be read with:
`docker-compose logs`
To stop:
`docker-compose down`

## Building the containter
A Dockerfile is provided within the repo. To build the container locally, run this command on the same directory the repo was cloned:
`sudo docker build -f Dockerfile . -t some_name/tracker:latest`
It will produce an image called `some_name/tracker`, which can be used in the Docker Compose section.

# Running from source as a service

For ease of maintenance, it is recommended to run the tracker as a service.

This is an example for running it under as the current user using `systemd`:
```
[Unit]
Description=Chihaya BT tracker
After=network.target
[Service]
Type=simple
#User=chihaya
#Group=chihaya
WorkingDirectory=/home/user/github/tracker
ExecStart=/home/user/github/tracker/chihaya --config dist/example_config.yaml
Restart=on-failure
[Install]
WantedBy=multi-user.target
```

To try it, change `/home/user/github/tracker` to where the code was cloned and run:
```bash=
mkdir -p ~/.config/systemd/user
# PASTE FILE IN ~/.config/systemd/user/tracker.service
systemctl --user enable tracker
systemctl --user start tracker
systemctl --user status tracker
```

## Contributing

Contributions to this project are welcome, encouraged, and compensated. For more details, please check [this](https://lbry.tech/contribute) link.

## License

LBRY's code changes are MIT licensed, and the upstream Chihaya code is licensed under a BSD 2-Clause license. For the full license, see [LICENSE](LICENSE).

## Security

We take security seriously. Please contact security@lbry.com regarding any security issues. [Our PGP key is here](https://lbry.com/faq/pgp-key) if you need it.

## Contact

The primary contact for this project is [@shyba](mailto:vshyba@lbry.com).
