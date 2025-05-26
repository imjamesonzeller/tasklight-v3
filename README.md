# Tasklight

**Tasklight** is a minimalist macOS app inspired by Spotlight, built to make task entry as fast and seamless as possible. Using a global hotkey, you can instantly open a lightweight input bar, type a natural language task like “Finish essay by Friday,” and have it automatically parsed and added to your Notion database.

---

## 🧠 Purpose

Tasklight was created to reduce the friction of capturing tasks. Rather than switching apps or losing focus, you can log tasks directly from anywhere on your system with just a keyboard shortcut. It’s perfect for fast-paced workflows and thought capture.

---

## ✨ Features

- Global hotkey to summon a Spotlight-style input window
- Transparent, always-on-top, distraction-free interface
- Natural language input processed with GPT (OpenAI)
- Automatically creates structured tasks in your connected Notion database
- Built-in settings window for updating your Notion/OpenAI configuration
- Secure local storage of secrets (using Apple Keychain)
- Notion OAuth integration — no manual token entry needed
- Select from your existing Notion databases in-app
- Keyboard-driven UI with instant show/hide and submission
- Clean, minimal UI with subtle animations

> **⚠️ AI Usage Disclaimer:**  
> Tasklight's AI-powered natural language parsing is currently **unavailable**. Support for AI task processing will return in a future update as part of a new system that includes **Bring Your Own Key (BYOK)** support and optional usage tiers. Stay tuned!

---

## ⚙️ Tech Stack

- **Wails v3** – Native macOS app framework (Go + Web)
- **Go** – Backend logic, Notion API, and global hotkey handling
- **React** – Frontend interface
- **OpenAI GPT-4o** – Parses task input into structured Notion-compatible data
- **Notion API** – For reading and writing task data
- **Apple Keychain** – Securely stores credentials locally
- **golang.design/x/hotkey** – Enables global macOS hotkey

---

## 💡 Inspiration

This project was inspired by [Coding With Lewis](https://youtu.be/lhjgj45x66Y?si=WroHyV6KREMvTNdW), who demonstrated a similar productivity concept. Tasklight builds on that foundation with added intelligence, Notion integration, and a refined user experience.

---

## 🚀 Usage

1. Press `Ctrl + Space` (or your configured shortcut) to launch the input window.
2. Type a task in natural language.
3. Press `Enter` to send it to your selected Notion database.
4. Press `Escape` to hide the window anytime.

---

## 📦 Configuration

Tasklight handles authentication and configuration directly within the app:

- 🔐 Secrets like your Notion integration token and OpenAI key are stored securely via Apple Keychain.
- 📚 Select your Notion database from a list after authenticating via OAuth.
- ⚙️ Configure your global hotkey and view current settings via the Settings window.

---

## 🔭 Future Plans

Planned improvements and additions:

- Usage quotas and optional tiered AI usage plans (e.g. 5 tasks/day free, unlimited with subscription)
- Offline fallback with local task queueing
- Recurring task parsing and smart tagging
- Optional dark/light mode toggle
- Task history viewer
- Integration with more platforms beyond Notion (e.g. Todoist, Google Tasks)

---

Made using Go, React, GPT, and a little obsession with clean interfaces.