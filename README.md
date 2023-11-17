[![GitHub Repo stars](https://img.shields.io/github/stars/nanreh/portpusher)](https://github.com/nanreh/portpusher "PortPusher Github Repo")

# PortPusher
PortPusher is a tiny program in a tiny Docker image that keeps your [Bittorrent](https://en.wikipedia.org/wiki/BitTorrent) client(s) updated with the forwarded port from your [Gluetun](https://github.com/qdm12/gluetun) VPN session. It's the perfect companion container for your homelab/seedbox.

Every few minutes, PortPusher checks Gluetun for the current forwarded port and "pushes" it to your Bittorrent client.

Supported BitTorrent clients:
* [Transmission](https://transmissionbt.com/)
* [QBittorrent](https://www.qbittorrent.org/)
* [Deluge](https://deluge-torrent.org/)

Sample "push" to three separate Bittorrent clients.
```bash
# docker logs -f portpusher
[Info] gluetun: Client ready host=localhost port=8000
[Info] transmission: Client ready host=localhost port=9091
[Info] qbittorrent: Client ready host=localhost port=8080
[Info] deluge: Client ready host=localhost port=8112
[Info] Running...
[Info] gluetun: Forwarded port is 54719
[Info] transmission: Pushing port 54719, current port is 51413
[Info] transmission: Port pushed
[Info] qbittorrent: Pushing port 54719, current port is 6881
[Info] qbittorrent: Port pushed
[Info] deluge: Pushing port 54719, current port is 6881
[Info] deluge: Port pushed
[Info] Done. Next push in 10m0s.
[Info] Running...
[Info] gluetun: Forwarded port is 54719
[Info] transmission: Port is correct
[Info] qbittorrent: Port is correct
[Info] deluge: Port is correct
[Info] Done. Next push in 10m0s.
```

## Setup

### docker-compose (recommended)

```yaml
---
  portpusher:
    image: nanreh/portpusher:latest
    container_name: portpusher
    network_mode: "service:vpn"
    environment:
      - TRANSMISSION_ENABLED=true
      - QBITTORRENT_ENABLED=true
      - DELUGE_ENABLED=true
```

### docker run

```bash
docker run -d \
  --name=portpusher \
  -e TRANSMISSION_ENABLED=true \
  -e QBITTORRENT_ENABLED=true \
  -e DELUGE_ENABLED=true \
  nanreh/portpusher:latest
```

## Parameters

PortPusher is configured using environment variables. Available values are:

| Parameter | Function |
| :----: | --- |
| `GLUETUN_HOST` | Gluetun hostname (default=localhost) |
| `GLUETUN_PORT` | Gluetun port (default=8000) |
| `PUSHER_LOG_LEVEL` | One of DEBUG, INFO, WARN, ERROR (default=INFO) |
| `PUSHER_DELAY_ERROR` | Minutes to wait until next push attempt after a push failue (default=5) |
| `PUSHER_DELAY_SUCCESS` | Minutes to wait until next push attempt after a successful push (default=10) |
| `TRANSMISSION_ENABLED` | Is Transmission enabled? (default=false) |
| `TRANSMISSION_HOST` | Transmission hostname (default=localhost) |
| `TRANSMISSION_PORT` | Transmission port (default=9091) |
| `TRANSMISSION_USER` | Transmission username (default=admin) |
| `TRANSMISSION_PASS` | Transmission password (default=password) |
| `QBITTORRENT_ENABLED` | Is QBittorrent enabled? (default=false) |
| `QBITTORRENT_HOST` | QBittorrent hostname (default=localhost) |
| `QBITTORRENT_PORT` | QBittorrent port (default=8080) |
| `QBITTORRENT_USER` | QBittorrent username (default=admin) |
| `QBITTORRENT_PASS` | QBittorrent password (default=adminadmin) |
| `DELUGE_ENABLED` | Is Deluge enabled? (default=false) |
| `DELUGE_HOST` | Deluge hostname (default=localhost) |
| `DELUGE_PORT` | Deluge port (default=8112) |
| `DELUGE_USER` | Deluge username (default=admin) |
| `DELUGE_PASS` | Deluge password (default=deluge) |

The architectures supported by this image are `amd64` and `arm64`.

## Port Forwarding Primer

When you're online, your public IP address is provided by your Internet service provider and assigned to some piece of hardware in your hand or in your home. Most of your online activity involves making requests to fetch content from other IP addresses, like when you fetch a web page or play a song on Spotify. To make such a request, you also need to specify a port number.

The defining feature of peer-to-peer systems like Bittorrent is that every peer in a "swarm" can send a request directly to any other peer. If one peer needs a chunk of data that another peer has, it simply makes a request to that peer and requests the data. Here again, making a request requires both the peer's IP address and a port number.

Your Bittorrent software needs to know the IP address and port number where it can be reached so it can share it with other peers. 

### VPN Port Forwarding

For Bittorrent to work while you're connected to a VPN, you need a VPN service that supports port forwarding. With port forwarding, your VPN provider will reserve a "listening" port just for you on the public IP address it provides. Other computers on the Internet can use that IP address and port number to send you requests and your VPN provider will funnel them to you.

Gluetun's logs will show when it negotiates a forwarded port with your VPN provider:
```
INFO [port forwarding] starting
INFO [port forwarding] Found saved forwarded port data for port 44201
INFO [port forwarding] Port forwarded data expires in 61 days
INFO [port forwarding] port forwarded is 44201
```

So Gluetun knows what your VPN listening port is but how do your Bittorrent clients learn about it? Each Bittorrent client has its own interface where you can manually provide the forwarded port:

| Client | Listening Port Location |
| :----: | :----: |
| Transmission | Edit preferences > Network > Peer Listening Port |
| QBittorrent | Tools > Options > Connection > Listening Port |
| Deluge | Preferences > Network > Incoming Address |

But this is tedious, fragile, and screams for automation... PortPusher is a simple program to automate it.

## Notes

* Not all VPN providers support port forwarding. Even those that do may not provide port forwarding in every region. So be sure to read the documentation for your provider. 
* Gluetun must be configured with `VPN_PORT_FORWARDING=on` so it requests port forwarding when it connects to the VPN provider (see the [test stack](./test_stack/README.md)).
* When no forwarded port is available, Gluetun will respond with port `0`. You may see this as the VPN is connecting and if it persists there is a problem with your VPN's port forwarding setup.
* PortPusher can't talk to Deluge until after the first time you log in to its web console.

## Test Stack

Included in this repo is a [test Docker Compose configuration](./test_stack/README.md) with Gluetun, Transmission, Deluge, QBittorrent, PortPusher, and [Docker Autoheal](https://github.com/willfarrell/docker-autoheal). This is a working Docker Compose stack that's a great reference of what a good setup looks like.
