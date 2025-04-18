import { useEffect, useState } from "react";
import { SettingsService as s, FrontendSettings } from "../bindings/github.com/imjamesonzeller/tasklight-v3";
import { ApplicationSettings } from "./types/ExtendedSettings.ts";

export default function Settings() {
    const [settings, setSettings] = useState<ApplicationSettings>({
        notion_db_id: "",
        use_open_ai: false,
        openai_api_key: "",
        notion_secret: "",
        theme: "light",
        launch_on_startup: false,
        hotkey: "ctrl+space",
        has_notion_secret: false,
        has_openai_key: false,
    });

    const [status, setStatus] = useState("");

    useEffect(() => {
        s.GetSettings()
            .then(({
                       notion_db_id,
                       use_open_ai,
                       theme,
                       launch_on_startup,
                       has_notion_secret,
                       has_openai_key,
                       hotkey
                   }: FrontendSettings) => {
                setSettings({
                    notion_db_id,
                    use_open_ai,
                    theme,
                    launch_on_startup,
                    has_notion_secret,
                    has_openai_key,
                    hotkey: hotkey,
                    notion_secret: "",
                    openai_api_key: ""
                });
            })
            .catch((err) => console.error("Failed to load settings", err));
    }, []);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        const { name, value, type } = e.target;

        const isCheckbox = type === "checkbox";
        const newValue = isCheckbox
            ? (e.target as HTMLInputElement).checked
            : value;

        setSettings((prev) => ({
            ...prev,
            [name]: newValue,
        }));
    };

    const handleSave = async () => {
        const payload = { ...settings };

        // Prevent overwriting stored secrets if the user didn’t modify them
        // @ts-ignore
        if (settings.notion_secret.trim() === "") {
            delete (payload as any).notion_secret;
        }
        // @ts-ignore
        if (settings.openai_api_key.trim() === "") {
            delete (payload as any).openai_api_key;
        }

        try {
            await s.UpdateSettings(payload);
            setStatus("✅ Settings saved.");
        } catch (err) {
            console.error(err);
            setStatus("❌ Failed to save settings.");
        }
    };

    return (
        <div className="p-4 flex flex-col gap-4">
            <h2 className="text-xl font-bold">Settings</h2>

            <div>
                <label className="block">Notion DB ID</label>
                <input name="notion_db_id" value={settings.notion_db_id} onChange={handleChange} className="w-full" />
            </div>

            <div>
                <label className="block">Notion Secret</label>
                <input
                    name="notion_secret"
                    value={settings.notion_secret}
                    onChange={handleChange}
                    className="w-full"
                    type="password"
                    placeholder={settings.has_notion_secret ? "••••••••••" : ""}
                />
                {settings.has_notion_secret && (
                    <span className="text-green-600 text-sm">✔ Key stored</span>
                )}
            </div>

            <div>
                <label className="block">Use OpenAI</label>
                <input name="use_open_ai" type="checkbox" checked={settings.use_open_ai} onChange={handleChange} />
            </div>

            <div>
                <label className="block">OpenAI API Key</label>
                <input
                    name="openai_api_key"
                    value={settings.openai_api_key}
                    onChange={handleChange}
                    className="w-full"
                    type="password"
                    placeholder={settings.has_openai_key ? "••••••••••" : ""}
                />
                {settings.has_openai_key && (
                    <span className="text-green-600 text-sm">✔ Key stored</span>
                )}
            </div>

            <div>
                <label className="block">Theme</label>
                <select name="theme" value={settings.theme} onChange={handleChange} className="w-full">
                    <option value="light">Light</option>
                    <option value="dark">Dark</option>
                    <option value="system">System</option>
                </select>
            </div>

            <div>
                <label className="block">Launch on Startup</label>
                <input name="launch_on_startup" type="checkbox" checked={settings.launch_on_startup} onChange={handleChange} />
            </div>

            <div>
                <label className="block">Hotkey</label>
                <input
                    name="hotkey"
                    type="text"
                    className="w-full"
                    value={settings.hotkey}
                    onChange={handleChange}
                />
            </div>

            <button className="mt-4 p-2 bg-blue-600 text-white rounded" onClick={handleSave}>
                Save
            </button>
            {status && <p className="text-sm">{status}</p>}
        </div>
    );
}