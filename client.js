const tls = require('tls');
const crypto = require('crypto');

function startClient(config) {
  let tlsSocket = null;
  let reconnectTimer = null;

  function connect() {
    console.log(`[CLIENT] → Connecting to ${config.serverHost}:${config.serverPort}...`);

    const options = {
      host: config.serverHost,
      port: config.serverPort,
      rejectUnauthorized: false,
      minVersion: 'TLSv1.2'
    };

    tlsSocket = tls.connect(options, () => {
      console.log(`[CLIENT] ✓ TLS connected to ${config.serverHost}:${config.serverPort}`);
      console.log(`[CLIENT] ✓ Cipher: ${tlsSocket.getCipher().name}`);
      
      // Send quantum auth pattern
      const authPattern = crypto.randomBytes(16);
      tlsSocket.write(authPattern);
      console.log(`[CLIENT] → Sent auth pattern: ${authPattern.toString('hex').substring(0, 16)}...`);
    });

    tlsSocket.on('data', (data) => {
      console.log(`[CLIENT] ← Received ${data.length} bytes`);
    });

    tlsSocket.on('error', (err) => {
      console.error(`[CLIENT] ✗ Error: ${err.message}`);
      scheduleReconnect();
    });

    tlsSocket.on('end', () => {
      console.log('[CLIENT] ✗ Connection closed');
      scheduleReconnect();
    });

    tlsSocket.on('close', () => {
      console.log('[CLIENT] ✗ Socket closed');
      scheduleReconnect();
    });
  }

  function scheduleReconnect() {
    if (reconnectTimer) return;
    
    console.log('[CLIENT] ⏳ Reconnecting in 5s...');
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      connect();
    }, 5000);
  }

  connect();

  return {
    getSocket: () => tlsSocket,
    disconnect: () => {
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (tlsSocket) tlsSocket.destroy();
    }
  };
}

module.exports = { startClient };
