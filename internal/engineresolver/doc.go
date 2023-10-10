// Package engineresolver contains the resolver used by the OONI engine. This
// resolver will try to figure out which is the best service for running
// domain name resolutions and will consistently use it.
//
// Occasionally this code will also swap the best resolver with other
// ~good resolvers to give them a chance to perform.
//
// The penalty/reward mechanism is strongly derivative, so the code should
// adapt ~quickly to changing network conditions. Occasionally, we will
// have longer resolutions when trying out other resolvers.
//
// At the beginning we randomize the known resolvers so that we do not
// have any preferential ordering. The initial resolutions may be slower
// if there are many issues with resolvers.
//
// The system resolver is given intermediate priority at the beginning (i.e.,
// 0.5) but it will of course be the most popular resolver if anything else
// is failing us. (We will still occasionally probe for other working
// resolvers and increase their score on success.)
//
// We also support a socks5 proxy. When such a proxy is configured,
// the code WILL skip http3 resolvers AS WELL AS the system
// resolver, in an attempt to avoid leaking your queries.
package engineresolver
