package archival

// DNSQueryType allows to access dnsQueryType from unit tests
type DNSQueryType = dnsQueryType

func (qtype dnsQueryType) IPOfType(addr string) bool {
	return qtype.ipoftype(addr)
}
