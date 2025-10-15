# Tasklight Privacy Notice

Last updated: 2024-10-15

Tasklight is a desktop application that runs on your Mac. This notice explains the data the app touches, how that data is handled, and the limited situations where it leaves your device.

## What We Store Locally

- **Application settings.** Preferences such as theme, chosen Notion database, and key bindings are saved to a `settings.json` file in your macOS user configuration directory.
- **Notion access token.** After you complete the Notion OAuth flow, the resulting token is stored in your macOS Keychain under the `com.tasklight.app` service. The token is never written to disk outside of the Keychain unless you explicitly launch Tasklight with `TASKLIGHT_SKIP_KEYCHAIN=1` for development purposes, in which case it is saved alongside `settings.json`.
- **OpenAI API key (optional).** If you enable Bring Your Own Key (BYOK), your key is saved in the Keychain. When BYOK is disabled or you clear credentials, the key is removed.
- **Task input.** Tasklight processes the text you type in memory only. It is not written to disk by the app.

## Notion OAuth Handshake

Tasklight uses a lightweight Tasklight-managed HTTPS service to broker the Notion OAuth exchange (`/tasklight/notion/oauth/*` on `api.jamesonzeller.com`). The service:

- Provides the Notion authorization URL.
- Accepts the short-lived `handoff` code returned by Notion and trades it for an access token.
- Returns the access token to the desktop app so it can be stored locally in your Keychain.

The service does not persist your Notion access token. Once relayed to the desktop client, the token remains on your device. No Notion database contents are sent to Tasklight infrastructure.

## Optional Parsing Service

Natural-language parsing is optional:

- **BYOK enabled.** Tasklight sends your task text directly to OpenAI via the official API using the key you supplied. No Tasklight-operated servers are involved in this path. OpenAI processes the text under its own terms of service.
- **BYOK disabled (default).** Tasklight calls the hosted parsing endpoint at `https://api.jamesonzeller.com/tasklight/parse`. The request contains:
  - The task text in the JSON body.
  - Your Notion bot ID in the `X-Notion-User-Id` header so the service can enforce quotas.

The hosted parser returns a structured task response and does not store or log the task contents or headers beyond what is necessary to fulfill the request and manage abuse safeguards. Notion access tokens, database payloads, and OpenAI keys are never transmitted to the hosted parser.

You can opt out of all remote parsing by enabling BYOK in settings or by disconnecting from the internet; task submission will fall back to sending the raw text to Notion if parsing fails.

## Data Sharing

- **Notion.** When you submit a task, Tasklight sends the structured payload and your stored Notion token directly to Notion's API over HTTPS.
- **OpenAI.** If you supply an OpenAI API key, Tasklight makes requests to OpenAI's API in your name. Those requests and responses are governed by OpenAI's privacy practices.
- **Tasklight infrastructure.** Limited to the OAuth helper and hosted parsing endpoints described above. We do not run analytics, crash reporting, or telemetry services in the desktop app.

## Security and Retention

- Credentials are stored using the macOS Keychain APIs with `AccessibleWhenUnlocked` protection.
- The settings file lives in `~/Library/Application Support/Tasklight/settings.json`. You may delete it at any time; Tasklight will recreate it with defaults.
- The **Clear Cache** action in settings removes stored credentials and preferences and disables launch-on-startup entries. You can also remove the Keychain entries named `NotionAccessToken` and `OpenAIAPISecret` manually using Keychain Access.

## Children

Tasklight is designed for general productivity use and is not directed at children under 13.

## Changes and Contact

We will update this notice when Tasklight's data flows change. Material updates will be published in this repository.

Questions or concerns? Open an issue at `https://github.com/imjamesonzeller/tasklight-v3/issues` and include "Privacy" in the title.
