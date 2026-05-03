class AIFeedback {
  constructor(aiEndpoint) {
    this.aiEndpoint = aiEndpoint;
    this.history = [];
  }

  async reportSuccess(protocol, port, latency) {
    this.history.push({
      protocol,
      port,
      latency,
      success: true,
      timestamp: Date.now()
    });

    console.log(`[AI-FEEDBACK] ✓ Success: ${protocol}:${port} (${latency}ms)`);
  }

  async reportFailure(protocol, port, reason) {
    this.history.push({
      protocol,
      port,
      reason,
      success: false,
      timestamp: Date.now()
    });

    console.log(`[AI-FEEDBACK] ✗ Failure: ${protocol}:${port} - ${reason}`);

    // ارسال به AI برای یادگیری
    if (this.history.length % 10 === 0) {
      await this.sendToAI();
    }
  }

  async sendToAI() {
    const prompt = `Learn from these connection attempts in Iran:
${JSON.stringify(this.history.slice(-20), null, 2)}

What patterns indicate DPI blocking? Suggest improvements.`;

    try {
      const response = await fetch(this.aiEndpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          messages: [{ role: 'user', content: prompt }],
          temperature: 0.7
        })
      });

      const result = await response.json();
      console.log('[AI-FEEDBACK] AI Insights:', result.choices[0].message.content);
    } catch (err) {
      console.error('[AI-FEEDBACK] Failed to send feedback:', err.message);
    }
  }
}

module.exports = { AIFeedback };
