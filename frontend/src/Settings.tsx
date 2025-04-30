import { useEffect, useState } from "react";
import { SettingsService as s } from "../bindings/github.com/imjamesonzeller/tasklight-v3/settingsservice";
import {
    DatabaseMinimal,
    NotionService as n
} from "../bindings/github.com/imjamesonzeller/tasklight-v3"

type SelectNotionDBProps = {
    databases: DatabaseMinimal[];
    value: string;
    onChange: (value: string) => void;
};

function SelectNotionDB({ databases, value, onChange }: SelectNotionDBProps) {
    const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
        onChange(event.target.value);
    };

    return (
        <div>
            <select value={value} onChange={handleChange}>
                <option value="" disabled>Select an option</option>
                {databases.map((db) => (
                    <option key={db.id} value={db.id}>
                        {db.title?.[0]?.text?.content || `Untitled (${db.id.slice(0, 6)}…)`}
                    </option>
                ))}
            </select>
        </div>
    );
}

export default function Settings() {
    const [settings, setSettings] = useState({
        notion_db_id: "",
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

    useEffect(() => {
        getNotionDBs()
    }, []);

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

    const [notionDBs, setNotionDBs] = useState<DatabaseMinimal[]>([]);

    const getNotionDBs = () => {
        n.GetNotionDatabases()
            .then((res) => setNotionDBs(res?.results ?? []))
    }

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
                <label>Notion Database:</label>
                <SelectNotionDB
                    databases={notionDBs}
                    value={settings.notion_db_id}
                    onChange={(value) =>
                        setSettings((prev) => ({ ...prev, notion_db_id: value }))
                    }
                />
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
