package portfiltering

//
// List of ports we want to measure
//

// List generated from nmap-services: https://github.com/nmap/nmap/blob/master/nmap-services
// Note: Using privileged ports like :80 requires elevated permissions
var Ports = []string{
	"80",    // tcp - World Wide Web HTTP
	"631",   // udp - Internet Printing Protocol
	"161",   // udp - Simple Net Mgmt Proto
	"137",   // udp - NETBIOS Name Service
	"123",   // udp - Network Time Protocol
	"138",   // udp - NETBIOS Datagram Service
	"1434",  // udp - Microsoft-SQL-Monitor
	"135",   // udp, tcp - epmap | Microsoft RPC services | DCE endpoint resolution
	"67",    // udp - DHCP/Bootstrap Protocol Server
	"23",    // tcp
	"53",    // udp, tcp - Domain Name Server
	"443",   // tcp - secure http (SSL)
	"21",    // tcp - File Transfer [Control]
	"22",    // tcp - Secure Shell Login
	"500",   // udp
	"68",    // udp - DHCP/Bootstrap Protocol Client
	"520",   // udp - router routed -- RIP
	"1900",  // udp - Universal PnP
	"25",    // tcp - Simple Mail Transfer
	"4500",  // udp - IKE Nat Traversal negotiation (RFC3947)
	"514",   // udp - BSD syslogd(8)
	"49152", // udp
	"162",   // udp - snmp-trap
	"69",    // udp - Trivial File Transfer
	"5353",  // udp - Mac OS X Bonjour/Zeroconf port
	"49154", // udp
	"3389",  // tcp - Microsoft Remote Display Protocol (aka ms-term-serv, microsoft-rdp) | MS WBT Server
	"110",   // tcp - PostOffice V.3 | Post Office Protocol - Version 3
	"1701",  // udp
	"998",   // udp
	"996",   // udp
	"997",   // udp
	"999",   // udp - Applix ac
	"3283",  // udp - Apple Remote Desktop Net Assistant reporting feature
	"49153", // udp
	"445",   // tcp - SMB directly over IP
	"1812",  // udp - RADIUS authentication protocol (RFC 2138)
	"136",   // udp - PROFILE Naming System
	"139",   // tcp, udp - NETBIOS Session Service
	"143",   // tcp - Interim Mail Access Protocol v2 | Internet Message Access Protocol
	"2222",  // udp - Microsoft Office OS X antipiracy network monitor
	"3306",  // tcp
	"2049",  // udp - networked file system
	"32768", // udp - OpenMosix Autodiscovery Daemon
	"5060",  // udp - Session Initiation Protocol (SIP)
	"8080",  // tcp - http-alt | Common HTTP proxy/second web server port | HTTP Alternate (see port 80)
	"1433",  // udp - Microsoft-SQL-Server
	"3456",  // udp - also VAT default data
	"1723",  // tcp - Point-to-point tunnelling protocol
	"111",   // tcp, udp - sunrpc | portmapper, rpcbind | SUN Remote Procedure Call
	"995",   // tcp - POP3 protocol over TLS/SSL | pop3 protocol over TLS/SSL (was spop3) | POP3 over TLS protocol
	"993",   // tcp - imap4 protocol over TLS/SSL | IMAP over TLS protocol
	"20031", // udp - BakBone NetVault primary communications port
	"1026",  // udp - Commonly used to send MS Messenger spam
	"7",     // udp
	"5900",  // tcp - rfb | Virtual Network Computer display 0 | Remote Framebuffer
	"1646",  // udp - radius accounting
	"1645",  // udp - radius authentication
	"593",   // udp # HTTP RPC Ep Map
	"1025",  // tcp, udp - blackjack | IIS, NFS, or listener RFS remote_file_sharing | network blackjack
	"518",   // udp - (talkd)
	"2048",  // udp
	"626",   // udp - Mac OS X Server serial number (licensing) daemon
}
