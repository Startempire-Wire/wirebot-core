import { Type } from "@sinclair/typebox";
import type { ClawdbotPluginApi } from "clawdbot/plugin-sdk";

type TwilioConfig = {
  accountSid: string;
  authToken: string;
  fromNumber: string;
  statusCallback?: string;
};

function resolveEnvVars(value: string): string {
  return value.replace(/\$\{([^}]+)\}/g, (_, envVar) => {
    const envValue = process.env[envVar];
    if (!envValue) throw new Error(`Environment variable ${envVar} is not set`);
    return envValue;
  });
}

function parseConfig(raw: unknown): TwilioConfig {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    throw new Error("sms-twilio config required");
  }
  const cfg = raw as Record<string, unknown>;
  if (typeof cfg.accountSid !== "string") throw new Error("accountSid required");
  if (typeof cfg.authToken !== "string") throw new Error("authToken required");
  if (typeof cfg.fromNumber !== "string") throw new Error("fromNumber required");
  return {
    accountSid: resolveEnvVars(cfg.accountSid),
    authToken: resolveEnvVars(cfg.authToken),
    fromNumber: resolveEnvVars(cfg.fromNumber),
    statusCallback: typeof cfg.statusCallback === "string" ? cfg.statusCallback : undefined,
  };
}

async function sendSms(cfg: TwilioConfig, to: string, message: string) {
  const url = `https://api.twilio.com/2010-04-01/Accounts/${cfg.accountSid}/Messages.json`;
  const body = new URLSearchParams({
    To: to,
    From: cfg.fromNumber,
    Body: message,
  });
  if (cfg.statusCallback) body.set("StatusCallback", cfg.statusCallback);

  const auth = Buffer.from(`${cfg.accountSid}:${cfg.authToken}`).toString("base64");
  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Basic ${auth}`,
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body,
  });
  const text = await res.text();
  if (!res.ok) throw new Error(`Twilio error ${res.status}: ${text}`);
  return text ? JSON.parse(text) : {};
}

const plugin = {
  id: "sms-twilio",
  name: "SMS (Twilio)",
  description: "Twilio SMS tool (skeleton; outbound only)",
  register(api: ClawdbotPluginApi) {
    const cfg = parseConfig(api.pluginConfig);
    api.logger.info("sms-twilio: registered (outbound only)");

    api.registerTool({
      name: "sms_send",
      label: "Send SMS (Twilio)",
      description: "Send an SMS via Twilio.",
      parameters: Type.Object({
        to: Type.String({ description: "Recipient phone (E.164)" }),
        message: Type.String({ description: "Message text" }),
      }),
      async execute(_toolCallId, params) {
        const { to, message } = params as { to: string; message: string };
        const result = await sendSms(cfg, to, message);
        return {
          content: [{ type: "text", text: `SMS sent to ${to}.` }],
          details: { result },
        };
      },
    });
  },
};

export default plugin;
