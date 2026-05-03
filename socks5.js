const net = require('net');

function startSocks5(config, getClientSocket) {
  const server = net.createServer((clientSocket) => {
    const clientAddr = `${clientSocket.remoteAddress}:${clientSocket.remotePort}`;
    console.log(`[SOCKS5] ✓ New connection from ${clientAddr}`);

    let targetSocket = null;

    clientSocket.once('data', (data) => {
      // SOCKS5 greeting
      if (data[0] !== 0x05) {
        console.log(`[SOCKS5] ✗ Invalid version from ${clientAddr}`);
        clientSocket.end();
        return;
      }

      // No auth required
      clientSocket.write(Buffer.from([0x05, 0x00]));

      clientSocket.once('data', (request) => {
        const cmd = request[1];
        const atyp = request[3];

        if (cmd !== 0x01) { // Only CONNECT supported
          clientSocket.write(Buffer.from([0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0]));
          clientSocket.end();
          return;
        }

        let host, port;
        
        if (atyp === 0x01) { // IPv4
          host = `${request[4]}.${request[5]}.${request[6]}.${request[7]}`;
          port = request.readUInt16BE(8);
        } else if (atyp === 0x03) { // Domain
          const len = request[4];
          host = request.slice(5, 5 + len).toString();
          port = request.readUInt16BE(5 + len);
        } else {
          clientSocket.write(Buffer.from([0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0]));
          clientSocket.end();
          return;
        }

        console.log(`[SOCKS5] → CONNECT ${host}:${port}`);

        // Connect to target
        targetSocket = net.connect(port, host, () => {
          console.log(`[SOCKS5] ✓ Connected to ${host}:${port}`);
          
          // Send success response
          clientSocket.write(Buffer.from([0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0]));
          
          // Pipe data
          clientSocket.pipe(targetSocket);
          targetSocket.pipe(clientSocket);
        });

        targetSocket.on('error', (err) => {
          console.error(`[SOCKS5] ✗ Target error ${host}:${port}: ${err.message}`);
          clientSocket.end();
        });

        targetSocket.on('close', () => {
          console.log(`[SOCKS5] ✗ Target closed ${host}:${port}`);
          clientSocket.end();
        });
      });
    });

    clientSocket.on('error', (err) => {
      console.error(`[SOCKS5] ✗ Client error ${clientAddr}: ${err.message}`);
      if (targetSocket) targetSocket.destroy();
    });

    clientSocket.on('close', () => {
      console.log(`[SOCKS5] ✗ Client closed ${clientAddr}`);
      if (targetSocket) targetSocket.destroy();
    });
  });

  server.listen(config.localPort, config.localHost, () => {
    console.log(`[SOCKS5] 🚀 Proxy listening on ${config.localHost}:${config.localPort}`);
  });

  server.on('error', (err) => {
    console.error(`[SOCKS5] ✗ Fatal error: ${err.message}`);
    process.exit(1);
  });

  return server;
}

module.exports = { startSocks5 };
