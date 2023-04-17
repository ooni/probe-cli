package model

// AddressFamily is a protocol address family.
type AddressFamily string

// AddressFamilyINET is the IPv4 protocol.
const AddressFamilyINET = AddressFamily("INET")

// AddressFamilyINET6 is the IPv6 protocol.
const AddressFamilyINET6 = AddressFamily("INET6")
