import { useEffect, useState } from "react";
import { SettingsService as s } from "../bindings/github.com/imjamesonzeller/tasklight-v3/settingsservice";
import { NotionService as n } from "../bindings/github.com/imjamesonzeller/tasklight-v3"

export default function Settings() {
    const [settings, setSettings] = useState({
        selected_notion_db_name: "",
        use_open_ai: false,
        theme: "light",
        launch_on_startup: false,
        hotkey: "ctrl+space",
        has_notion_secret: false,
        has_openai_key: false,
    });

    const [status, setStatus] = useState("");

    useEffect(() => {
        s.GetSettings()
            .then((res) => setSettings(res))
            .catch((err) => setStatus("Failed to load settings: " + err.message));
    }, []);

    // TODO: Add drop down selector to select database

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        const { name, value, type } = e.target;

        setSettings((prev) => ({
            ...prev,
            [name]: type === "checkbox"
                ? (e.target as HTMLInputElement).checked
                : value,
        }));
    };

    const saveSettings = () => {
        s.UpdateSettingsFromFrontend(settings)
            .then(() => setStatus("✅ Settings saved."))
            .catch((err) => setStatus("❌ Failed to save settings: " + err.message));
    };

    const connectNotion = () => {
        n.StartOAuth()
            .then(() => setStatus("🔗 Notion connected."))
            .catch((err) => setStatus("Failed to connect Notion: " + err.message));
    };

    return (
        <div className="p-4 space-y-4">
            <h1 className="text-xl font-bold">Settings</h1>
            <div>
                <label>Theme:</label>
                <select name="theme" value={settings.theme} onChange={handleChange}>
                    <option value="light">Light</option>
                    <option value="dark">Dark</option>
                </select>
            </div>

            <div>
                <label>Launch on Startup:</label>
                <input type="checkbox" name="launch_on_startup" checked={settings.launch_on_startup} onChange={handleChange} />
            </div>

            <div>
                <label>Hotkey:</label>
                <input type="text" name="hotkey" value={settings.hotkey} onChange={handleChange} />
            </div>

            <div>
                <label>Use OpenAI:</label>
                <input type="checkbox" name="use_open_ai" checked={settings.use_open_ai} onChange={handleChange} />
            </div>

            <div>
                <label>Connected to Notion:</label>
                <span>{settings.has_notion_secret ? "✅" : "❌"}</span>
                <button onClick={connectNotion} className="ml-2 px-2 py-1 bg-blue-500 text-white rounded">
                    Connect Notion
                </button>
            </div>

            <div>
                <label>Connected to OpenAI:</label>
                <span>{settings.has_openai_key ? "✅" : "❌"}</span>
            </div>

            <button onClick={saveSettings} className="mt-4 px-4 py-2 bg-green-600 text-white rounded">
                Save Settings
            </button>

            {status && <p className="mt-2 text-sm text-gray-700">{status}</p>}
        </div>
    );
}
