// Discord Audit Extension - logs all interactions to Wirebot scoreboard
const SCOREBOARD_URL = process.env.SCOREBOARD_URL || "http://127.0.0.1:8100";
const SCOREBOARD_TOKEN = process.env.SCOREBOARD_TOKEN || "65b918ba-baf5-4996-8b53-6fb0f662a0c3";

export default function discordAudit(api) {
  console.log("[discord-audit] Extension loaded");

  // Hook into message completion events
  api.on("message_complete", async (event) => {
    // Only log Discord messages
    if (!event.channel?.startsWith("discord")) return;
    
    const { message, response, session, timing, model, usage } = event;
    
    // Determine mode based on channel config
    let mode = "guest";
    const channelId = session?.channelId || "";
    const guildId = session?.guildId || "";
    
    // Check if sovereign channel (wirebot-comms)
    if (channelId === "1468200674545238191") {
      mode = "sovereign";
    } else if (guildId === "1058318315564576848") {
      mode = "community";
    }
    
    const interaction = {
      interaction_id: `discord-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      guild_id: guildId,
      guild_name: session?.guildName || "",
      channel_id: channelId,
      channel_name: session?.channelName || "",
      user_id: message?.author?.id || "",
      user_name: message?.author?.username || message?.author?.tag || "unknown",
      user_message: message?.content || "",
      bot_response: response?.content || "",
      response_time_ms: timing?.totalMs || 0,
      mode: mode,
      tools_used: response?.toolCalls?.map(t => t.name) || [],
      model: model || "",
      tokens_in: usage?.promptTokens || 0,
      tokens_out: usage?.completionTokens || 0
    };

    try {
      const resp = await fetch(`${SCOREBOARD_URL}/v1/discord/interaction`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${SCOREBOARD_TOKEN}`
        },
        body: JSON.stringify(interaction)
      });
      
      if (resp.ok) {
        console.log(`[discord-audit] Logged ${mode} interaction from ${interaction.user_name}`);
      } else {
        console.error(`[discord-audit] Failed to log: ${resp.status}`);
      }
    } catch (err) {
      console.error(`[discord-audit] Error: ${err.message}`);
    }
  });

  // Register /score command
  api.registerCommand?.("score", {
    description: "Check your Wirebot score",
    handler: async (args, ctx) => {
      try {
        const resp = await fetch(`${SCOREBOARD_URL}/v1/score`, {
          headers: { "Authorization": `Bearer ${SCOREBOARD_TOKEN}` }
        });
        const data = await resp.json();
        
        return `⚡ **Your Score: ${data.total || 0}/100**\n` +
               `📦 Ship: ${data.ship || 0}/40\n` +
               `📢 Distribution: ${data.distribution || 0}/25\n` +
               `💰 Revenue: ${data.revenue || 0}/20\n` +
               `🔧 Systems: ${data.systems || 0}/15\n` +
               `🔥 Streak: ${data.streak || 0} days`;
      } catch (err) {
        return `Error fetching score: ${err.message}`;
      }
    }
  });

  console.log("[discord-audit] Commands registered");
}
