// Signal-Desktop's own HTTPS fetches (updater, WebAPI's agent) as ts/util/createHTTPSAgent.node.ts
// sets them up: node:https Agent, tls.connect({...options, ca, servername}), Signal's RSA root
// only, no ALPN, Electron's default TLS (BoringSSL). Run with ELECTRON_RUN_AS_NODE=1:
//   electron desktop-fetch.js URL
const https = require('node:https');
const fs = require('node:fs');
const der = fs.readFileSync(process.env.SIGNAL_CA || 'signal.cer');
const ca = '-----BEGIN CERTIFICATE-----\n' + der.toString('base64').match(/.{1,64}/g).join('\n') + '\n-----END CERTIFICATE-----\n';
const url = new URL(process.argv[2] || 'https://updates2.signal.org/desktop/latest-linux.yml');
const agent = new https.Agent({ ca, keepAlive: false });
const t0 = Date.now();
https.get({ host: url.hostname, path: url.pathname, servername: url.hostname, ca, agent }, (res) => {
  console.log(`electron ${process.versions.electron}: ${Date.now() - t0} ms ${url.hostname} ${res.statusCode} ${res.socket.getProtocol()}`);
  res.resume(); res.on('end', () => process.exit(0));
}).on('error', (e) => { console.log('error', e.message); process.exit(1); });
