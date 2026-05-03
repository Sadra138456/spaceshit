const tls = require('tls');
const fs = require('fs');
const crypto = require('crypto');

function startServer(config) {
  const options = {
    key: fs.readFileSync(config.tlsKey),
    cert: fs.readFileSync(config.tlsCert),
    requestCert: false,
    rejectUnauthorized: false,
    minVersion: 'TLSv1.2'
  };

  const clients = new Map();

  const server = tls.createServer(options, (socket) => {
    const clientId = `${socket.remoteAddress}:${socket.remotePort}`;
    console.log(`[SERVER] ✓ Client connected: ${clientId}`);
    
    clients.set(clientId, socket);

    // Quantum pattern authentication
    socket.once('data', (data) => {
      if (data.length < 16) {
        console.log(`[SERVER] ✗ Invalid auth from ${clientId}`);
        socket.destroy();
        return;
      }

      const pattern = data.slice(0, 16).toString('hex');
      console.log(`[SERVER] ✓ Auth pattern: ${pattern.substring(0, 8)}...`);
      
      // Send ACK
      const ack = crypto.randomBytes(16);
      socket.write(ack);

      // Handle data tunneling
      socket.on('data', (chunk) => {
        console.log(`[SERVER] ← Received ${chunk.length} bytes from ${clientId}`);
        // Here you would forward to actual destination
        socket.write(Buffer.from([0x00])); // ACK
      });
    });

    socket.on('error', (err) => {
      console.error(`[SERVER] ✗ Error ${clientId}: ${err.message}`);
      clients.delete(clientId);
    });

    socket.on('end', () => {
      console.log(`[SERVER] ✗ Client disconnected: ${clientId}`);
      clients.delete(clientId);
    });
  });

  server.listen(config.serverPort, config.serverHost, () => {
    console.log(`[SERVER] 🚀 Listening on ${config.serverHost}:${config.serverPort}`);
  });

  server.on('error', (err) => {
    console.error(`[SERVER] ✗ Fatal error: ${err.message}`);
    process.exit(1);
  });

  return server;
}

module.exports = { startServer };
