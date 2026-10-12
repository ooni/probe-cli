# Google roots for the Google fronts

The roots libsignal v0.105.0 pins for its Google fronts (`rust/net/res/`), used by the
`reflector_g` and `android_g` circumvention routes to verify the front's certificate, as the apps
do. Copied byte for byte; SHA-256 of the DER:

| File | Certificate | SHA-256 |
|---|---|---|
| `GTSR1.cer` | GTS Root R1 | `2a575471e31340bc21581cbd2cf13e158463203ece94bcf9d3cc196bf09a5472` |
| `GTSR2.cer` | GTS Root R2 | `c45d7bb08e6d67e62e4235110b564e5f78fd92ef058c840aea4e6455d7585c60` |
| `GTSR3.cer` | GTS Root R3 | `15d5b8774619ea7d54ce1ca6d0b0c403e037a917f131e8a04e1e6b7a71babce5` |
| `GTSR4.cer` | GTS Root R4 | `71cca5391f9e794b04802530b363e121da8a3043bb26662fea4dca7fc951a4bd` |
| `GSR2.cer` | GlobalSign Root CA - R2 (expired 2021-12-15) | `ca42dd41745fd0b81eb902362cf9d8bf719da1bd1b1efc946f5b4c99f42c1b9e` |
| `GSR4.cer` | GlobalSign ECC Root CA - R4 | `bec94911c2955676db6c0a550986d76e3ba005667c442c9762b4fbb773de228c` |
| `GIAG2.cer` | Google Internet Authority G2 (expired 2017-12-31) | `9f630426df1d8abfd80ace98871ba833ab9742cb34838de2b5285ed54c0c7dcc` |

Go's verifier does not chain to the two expired roots; they are kept as libsignal ships them.
Refresh this directory when libsignal changes `rust/net/res/`. The Fastly fronts are verified
with Mozilla's roots (netxlite), Signal's hosts with `hellos/signal*.cer` or the `SignalCA` option.
