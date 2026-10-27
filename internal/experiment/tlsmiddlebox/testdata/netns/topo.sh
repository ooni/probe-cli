#!/bin/sh
# Builds a chain of network namespaces with real kernel routers, then runs "$@" in the probe's:
#
#   probe 10.0.1.2 -- r1 -- r2 -- r3 -- server 10.0.4.2:443 (server.py)
#
# A packet sent with TTL n expires at router n (ICMP time exceeded), and reaches the server
# from TTL 4. netns_test.go puts its middlebox rules on r2, so they see packets with TTL >= 3.
# Needs root: run.sh runs it in a privileged container, CI with sudo. Uses nsenter rather than
# `ip netns exec`, which remounts /sys and rootless podman refuses that.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
at() { ns=$1; shift; nsenter --net=/run/netns/"$ns" "$@"; }

for ns in probe r1 r2 r3 server; do ip netns add $ns; at $ns ip link set lo up; done
trap 'kill $(cat /run/netns-server.pid 2>/dev/null) 2>/dev/null; for ns in probe r1 r2 r3 server; do ip netns del $ns; done' EXIT

# link i joins two neighbours on 10.0.i.0/24 (veth a<i> on the left, b<i> on the right)
link() {  # link <i> <left> <right> <left ip> <right ip>
  ip link add a$1 netns $2 type veth peer name b$1 netns $3
  at $2 ip addr add $4/24 dev a$1; at $2 ip link set a$1 up
  at $3 ip addr add $5/24 dev b$1; at $3 ip link set b$1 up
}
link 1 probe r1 10.0.1.2 10.0.1.1
link 2 r1 r2 10.0.2.2 10.0.2.1
link 3 r2 r3 10.0.3.2 10.0.3.1
link 4 r3 server 10.0.4.1 10.0.4.2

at probe ip route add default via 10.0.1.1
at server ip route add default via 10.0.4.1
at r1 ip route add default via 10.0.2.1
at r2 ip route add 10.0.1.0/24 via 10.0.2.2
at r2 ip route add 10.0.4.0/24 via 10.0.3.1
at r3 ip route add default via 10.0.3.2
for r in r1 r2 r3; do
  # icmp_ratelimit=0: real routers rate-limit ICMP, but a test wants one answer per probe
  at $r sysctl -q -w net.ipv4.ip_forward=1 net.ipv4.icmp_ratelimit=0
done

at server python3 "$HERE/server.py" &
echo $! > /run/netns-server.pid
at probe python3 -c 'import socket, time
for _ in range(100):
    try: socket.create_connection(("10.0.4.2", 443), 1).close(); break
    except OSError: time.sleep(0.1)
else: raise SystemExit("server.py did not start")'
at probe "$@"
