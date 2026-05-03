import { startServer } from './server.js';
import { startClient } from './client.js';
import { startSOCKS5 } from './socks5.js';
import { startQuantumShaper } from './quantum-shaper.js';
import { startAIFeedback } from './ai-feedback.js';

const mode = process.argv[2] || 'both';

console.log(`[SPACESHIT] Starting in ${mode} mode...`);

switch (mode) {
    case 'server':
        startServer();
        startAIFeedback();
        break;

    case 'client':
        startSOCKS5();
        startQuantumShaper();
        startClient();
        break;

    case 'both':
        startServer();
        startAIFeedback();
        startSOCKS5();
        startQuantumShaper();
        startClient();
        break;

    default:
        console.log('Usage: node main.js [server|client|both]');
        process.exit(1);
}
