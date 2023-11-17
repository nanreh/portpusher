# Test Stack

This Docker Compose configuration shows how you should configure Gluetun and a few containers that depend on it for their networking (in this case Bittorrent clients Transmission, Deluge, and QBittorrent).

While this topic is covered in detail elsewhere, my own personal experience tells me that a standalone working configuration will be useful.

## Running

* Edit the `docker-compose.yml` file to include your actual VPN provider and credentials.
* Run the `init` script in this directory to create the directories containers will map as volumes.
* Run `docker-compose up -d`

## Notes

Except for the `gluetun` container, all containers are set with `network_mode: "service:vpn"`. This means their networking is provided by the `vpn` container and so all of their network traffic will happen on the VPN connection.

We define a `vpn` network and place the `vpn` container in it.

We map several ports from the `vpn` container to our host so that we can access the web interfaces of the containers in our test stack:

* [Deluge](http://localhost:8112/) - default password `deluge`
* [Transmission](http://localhost:9091/transmission/web/) - default login `admin/password`
* [QBittorrent](http://localhost:8080/) - default login `admin/adminadmin`

Also running is PortPusher. You can inspect its logs like any other Docker container with `docker logs -f portpusher`.

Each container is configured with a reasonable healthcheck and an `autoheal` container is also configured to restart containers that end up unhealthy.