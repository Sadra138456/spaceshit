const { startServer } = require('./server.js');
const { startClient } = require('./client.js');
const { startSocks5 } = require('./socks5.js');

const config = {
  serverHost: '0.0.0.0',
  serverPort: 443,
  localHost: '0.0.0.0',
  localPort: 1080,
  tlsKey: 'server.key',
  tlsCert: 'server.crt'
};

const mode = process.argv[2] || 'both';

console.log(`
╔═══════════════════════════════════════╗
║     SPACESHIT QUANTUM TUNNEL v1.0     ║
║     Anti-DPI Proxy System             ║
╚═══════════════════════════════════════╝
`);

if (mode === 'server' || mode === 'both') {
  console.log('[MAIN] Starting SERVER mode...');
  startServer(config);
}

if (mode === 'client' || mode === 'both') {
  console.log('[MAIN] Starting CLIENT mode...');
  
  setTimeout(() => {
    const client = startClient(config);
    startSocks5(config, () => client.getSocket());
  }, 1000);
}

console.log(`[MAIN] ✓ Running in ${mode.toUpperCase()} mode\n`);
