import OpenAI from "openai";

const client = new OpenAI({
    apiKey: "YOUR_API_KEY",
    baseURL: "https://api.gapgpt.app/v1"
});

let feedbackHistory = [];

export async function sendFeedback(result) {
    feedbackHistory.push({
        timestamp: Date.now(),
        ...result
    });

    // Keep last 100 results
    if (feedbackHistory.length > 100) {
        feedbackHistory.shift();
    }

    try {
        await client.responses.create({
            model: "gapgpt-qwen-3.5",
            input: `
Feedback from connection attempt:
${JSON.stringify(result)}

Recent history (last 10):
${JSON.stringify(feedbackHistory.slice(-10))}

Learn from this and improve your quantum strategy.
            `
        });

        console.log('[FEEDBACK] ✓ Sent to AI');

    } catch (err) {
        console.log('[FEEDBACK] ✗ Failed:', err.message);
    }
}

export function startAIFeedback() {
    console.log('[FEEDBACK] AI learning loop started');
    
    // Periodic summary
    setInterval(() => {
        const recent = feedbackHistory.slice(-10);
        const successCount = recent.filter(r => r.success).length;
        console.log(`[FEEDBACK] Success rate: ${successCount}/10`);
    }, 60000);
}
