// This file contains helpers for describing blocking rules.

// hijackPopularDNSServers returns an object containing the rules
// for hijacking popular DNS servers with `miniooni --censor`.
//
// This function is an helper function for populating test cases.
export function hijackPopularDNSServers() {
    return {
        // cloudflare
        "1.1.1.1:53/udp": "hijack-dns",
        "1.0.0.1:53/udp": "hijack-dns",
        // google
        "8.8.8.8:53/udp": "hijack-dns",
        "8.8.4.4:53/udp": "hijack-dns",
        // quad9
        "9.9.9.9:53/udp": "hijack-dns",
        "9.9.9.10:53/udp": "hijack-dns",
    }
}
