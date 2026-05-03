const net = require('net');
const { AdaptiveClient } = require('./adaptive-client');

class SOCKS5Server {
  constructor(port, serverHost, psk, aiEndpoint) {
    this.port = port;
    this.serverHost = serverHost;
    this.psk = psk;
    this.aiEndpoint = aiEndpoint;
    this.server = null;
  }

  start() {
    this.server = net.createServer((clientSocket) => {
      console.log('[SOCKS5] New client connection');
      this.handleClient(clientSocket);
    });

    this.server.listen(this.port, '127.0.0.1', () => {
      console.log(`[SOCKS5] ✓ Listening on 127.0.0.1:${this.port}`);
    });

    this.server.on('error', (err) => {
      console.error('[SOCKS5] Server error:', err.message);
    });
  }

  async handleClient(clientSocket) {
    let tunnelClient = null;

    // SOCKS5 handshake
    clientSocket.once('data', async (data) => {
      if (data[0] !== 0x05) {
        clientSocket.end();
        return;
      }

      // No authentication
      clientSocket.write(Buffer.from([0x05, 0x00]));

      clientSocket.once('data', async (data) => {
        if (data[0] !== 0x05 || data[1] !== 0x01) {
          clientSocket.end();
          return;
        }

        const addrType = data[3];
        let targetHost, targetPort;

        if (addrType === 0x01) {
          // IPv4
          targetHost = `${data[4]}.${data[5]}.${data[6]}.${data[7]}`;
          targetPort = data.readUInt16BE(8);
        } else if (addrType === 0x03) {
          // Domain
          const domainLen = data[4];
          targetHost = data.slice(5, 5 + domainLen).toString();
          targetPort = data.readUInt16BE(5 + domainLen);
        } else {
          clientSocket.end();
          return;
        }

        console.log(`[SOCKS5] Request: ${targetHost}:${targetPort}`);

        // Success response
        clientSocket.write(Buffer.from([
          0x05, 0x00, 0x00, 0x01,
          0x00, 0x00, 0x00, 0x00,
          0x00, 0x00
        ]));

        // اتصال به سرور از طریق adaptive tunnel
        try {
          tunnelClient = new AdaptiveClient(
            this.serverHost,
            this.psk,
            this.aiEndpoint
          );

          await tunnelClient.connect();

          // ارسال درخواست اتصال به سرور
          const connectRequest = Buffer.concat([
            Buffer.from([addrType]),
            Buffer.from(targetHost),
            Buffer.allocUnsafe(2)
          ]);
          connectRequest.writeUInt16BE(targetPort, connectRequest.length - 2);

          await tunnelClient.send(connectRequest);

          // پایپ کردن داده‌ها
          clientSocket.on('data', async (data) => {
            try {
              await tunnelClient.send(data);
            } catch (err) {
              console.error('[SOCKS5] Send error:', err.message);
              clientSocket.end();
            }
          });

          tunnelClient.onData = (data) => {
            clientSocket.write(data);
          };

          clientSocket.on('end', () => {
            tunnelClient.close();
          });

          clientSocket.on('error', (err) => {
            console.error('[SOCKS5] Client socket error:', err.message);
            tunnelClient.close();
          });

        } catch (err) {
          console.error('[SOCKS5] Tunnel connection failed:', err.message);
          clientSocket.end();
        }
      });
    });
  }

  stop() {
    if (this.server) {
      this.server.close();
      console.log('[SOCKS5] ✓ Server stopped');
    }
  }
}

function startSOCKS5(port, serverHost, psk, aiEndpoint) {
  const server = new SOCKS5Server(port, serverHost, psk, aiEndpoint);
  server.start();
  return server;
}

module.exports = { startSOCKS5, SOCKS5Server };
