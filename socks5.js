import net from 'net';
import { quantumConnect } from './quantum-shaper.js';

export function startSOCKS5() {
    const server = net.createServer(async (socket) => {
        socket.once('data', async (data) => {
            // SOCKS5 handshake
            if (data[0] !== 0x05) {
                socket.end();
                return;
            }

            // No auth required
            socket.write(Buffer.from([0x05, 0x00]));

            socket.once('data', async (data) => {
                if (data[0] !== 0x05 || data[1] !== 0x01) {
                    socket.end();
                    return;
                }

                // Parse target
                let target, port;
                const addrType = data[3];

                if (addrType === 0x01) {
                    // IPv4
                    target = `${data[4]}.${data[5]}.${data[6]}.${data[7]}`;
                    port = data.readUInt16BE(8);
                } else if (addrType === 0x03) {
                    // Domain
                    const domainLen = data[4];
                    target = data.slice(5, 5 + domainLen).toString();
                    port = data.readUInt16BE(5 + domainLen);
                } else {
                    socket.end();
                    return;
                }

                console.log(`[SOCKS5] Request: ${target}:${port}`);

                try {
                    // Connect via quantum client
                    const remote = await quantumConnect(target, port);

                    // Success response
                    socket.write(Buffer.from([0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0]));

                    // Relay data
                    socket.pipe(remote);
                    remote.pipe(socket);

                    socket.on('error', () => remote.end());
                    remote.on('error', () => socket.end());

                } catch (err) {
                    console.log('[SOCKS5] Connection failed:', err.message);
                    socket.write(Buffer.from([0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0]));
                    socket.end();
                }
            });
        });
    });

    server.listen(1080, '0.0.0.0', () => {
        console.log('[SOCKS5] Listening on 0.0.0.0:1080');
    });
}
