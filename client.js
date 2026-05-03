import tls from 'tls';
import { getQuantumStrategy } from './quantum-shaper.js';
import { sendFeedback } from './ai-feedback.js';

const SERVER_HOST = '185.208.172.162';
const SERVER_PORT = 443;

export function startClient() {
    console.log('[CLIENT] Starting quantum TLS client...');
    connectWithQuantum();
}

async function connectWithQuantum() {
    while (true) {
        try {
            console.log('[CLIENT] Connecting to server...');

            // Get quantum strategy from AI
            const strategy = await getQuantumStrategy({
                target: `${SERVER_HOST}:${SERVER_PORT}`,
                lastAttempt: Date.now()
            });

            console.log('[CLIENT] Quantum strategy:', strategy);

            // Apply quantum delay
            await sleep(strategy.delay_ms);

            // Connect with TLS
            const socket = tls.connect({
                host: SERVER_HOST,
                port: SERVER_PORT,
                servername: strategy.sni || 'www.google.com',
                rejectUnauthorized: false,
                minVersion: 'TLSv1.2',
                maxVersion: 'TLSv1.3',
                ciphers: strategy.ciphers || 'TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384'
            });

            socket.on('secureConnect', () => {
                console.log('[CLIENT] ✓ Connected successfully!');
                sendFeedback({ success: true, strategy });

                // Keep alive with quantum pattern
                setInterval(() => {
                    if (!socket.destroyed) {
                        socket.write(Buffer.from('ping'));
                    }
                }, 30000);
            });

            socket.on('error', (err) => {
                console.log('[CLIENT] ✗ Connection failed:', err.message);
                sendFeedback({ success: false, error: err.message, strategy });
            });

            socket.on('end', () => {
                console.log('[CLIENT] Connection closed');
            });

            // Wait before retry
            await sleep(10000);

        } catch (err) {
            console.log('[CLIENT] Error:', err.message);
            await sleep(5000);
        }
    }
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}
