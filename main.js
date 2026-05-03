const { startServer } = require('./server');
const { startSOCKS5 } = require('./socks5');

const CONFIG = {
  SERVER_PORT: 443,
  SOCKS5_PORT: 1080,
  PSK: 'ultra-secret-quantum-key-2026',
  AI_ENDPOINT: 'https://gapgpt.app/api/chat'
};

async function main() {
  const mode = process.argv[2] || 'both';
  const serverHost = process.argv[3] || '185.208.172.162';

  console.log('╔════════════════════════════════════════╗');
  console.log('║   🚀 Anti-DPI Intelligent Tunnel 🚀   ║');
  console.log('║   AI-Powered Censorship Bypass        ║');
  console.log('╚════════════════════════════════════════╝\n');

  if (mode === 'server' || mode === 'both') {
    console.log('[MAIN] Starting server mode...');
    const server = startServer(CONFIG.SERVER_PORT, CONFIG.PSK);
    
    process.on('SIGINT', () => {
      console.log('\n[MAIN] Shutting down server...');
      server.stop();
      process.exit(0);
    });
  }

  if (mode === 'client' || mode === 'both') {
    console.log('[MAIN] Starting client mode...');
    console.log(`[MAIN] Server: ${serverHost}`);
    
    const socks5 = startSOCKS5(
      CONFIG.SOCKS5_PORT,
      serverHost,
      CONFIG.PSK,
      CONFIG.AI_ENDPOINT
    );

    console.log('\n╔════════════════════════════════════════╗');
    console.log('║  ✓ SOCKS5 Proxy Ready!                ║');
    console.log(`║  📱 Mobile: ${serverHost}:${CONFIG.SOCKS5_PORT}     ║`);
    console.log('║  🌐 Browser: 127.0.0.1:1080           ║');
    console.log('╚════════════════════════════════════════╝\n');

    process.on('SIGINT', () => {
      console.log('\n[MAIN] Shutting down client...');
      socks5.stop();
      process.exit(0);
    });
  }

  console.log('[MAIN] ✓ All systems operational\n');
}

main().catch(err => {
  console.error('[MAIN] Fatal error:', err);
  process.exit(1);
});
