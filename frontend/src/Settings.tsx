import { useEffect, useState } from "react";
import { SettingsService as s } from "../bindings/github.com/imjamesonzeller/tasklight-v3/settingsservice";
import {
    DatabaseMinimal,
    NotionService as n
} from "../bindings/github.com/imjamesonzeller/tasklight-v3"
import "../public/settings.css";

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
        date_property_id: "",
        date_property_name: "",
    });

    const [status, setStatus] = useState("");
    const [notionDBs, setNotionDBs] = useState<DatabaseMinimal[]>([]);
    const [hasMultipleDateProps, setHasMultipleDateProps] = useState(false);
    const [dateValid, setDateValid] = useState(true);

    useEffect(() => {
        s.GetSettings()
            .then((res) => setSettings(res))
            .catch((err) => setStatus("Failed to load settings: " + err.message));
    }, []);

    useEffect(() => {
        getNotionDBs()
    }, []);

    useEffect(() => {
        const selected = notionDBs.find((db) => db.id === settings.notion_db_id);
        if (!selected) return;

        const dateProps = Object.entries(selected.properties ?? {}).filter(
            ([_, prop]) => prop.type === "date"
        );

        if (dateProps.length > 1) {
            setHasMultipleDateProps(true);
            // mark invalid until user selects one
            setDateValid(settings.date_property_id !== "");
        } else {
            setHasMultipleDateProps(false);

            if (dateProps.length === 1) {
                const [id, prop] = dateProps[0];
                setSettings((prev) => ({
                    ...prev,
                    date_property_id: id,
                    date_property_name: prop.name,
                }));
                setDateValid(true);
            } else {
                setSettings((prev) => ({
                    ...prev,
                    date_property_id: "",
                    date_property_name: "",
                }));
                setDateValid(false);
            }
        }
    }, [settings.notion_db_id, notionDBs, settings.date_property_id]);

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

    const connectNotion = async () => {
        try {
            setStatus("🔄 Connecting to Notion...");
            await n.StartOAuth();

            // Timeout to allow for settings save
            setTimeout(async () => {
                try {
                    const updatedSettings = await s.GetSettings();
                    setSettings(updatedSettings);

                    const dbResponse = await n.GetNotionDatabases();
                    setNotionDBs(dbResponse?.results ?? []);

                    setStatus("✅ Notion connected and databases loaded.");
                } catch (err: any) {
                    setStatus("❌ Failed to refresh after Notion connect: " + err.messages);
                }
            }, 1500)
        } catch (err:any) {
            setStatus("❌ Failed to connect Notion: " + err.messages);
        }
    }

    const getNotionDBs = async () => {
        const res = await n.GetNotionDatabases();
        const results = res?.results ?? [];

        setNotionDBs(results);

        const selected = results.find((db) => db.id === settings.notion_db_id);
        if (selected?.has_multiple_date_props) {
            setHasMultipleDateProps(true);
        } else {
            setHasMultipleDateProps(false);
        }
    };

    return (
        <div className="settings-container">
            <h1 className="settings-title">Settings</h1>

            <div>
                <label>Theme:</label>
                <select name="theme" value={settings.theme} onChange={handleChange}>
                    <option value="light">Light</option>
                    <option value="dark">Dark</option>
                </select>
            </div>

            <div>
                <label>Launch on Startup:</label>
                <input
                    type="checkbox"
                    name="launch_on_startup"
                    checked={settings.launch_on_startup}
                    onChange={handleChange}
                />
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
                <button onClick={connectNotion} className="notion-connect-btn">
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
                <label>Date Property:</label>
                {hasMultipleDateProps ? (
                    <select
                        value={settings.date_property_id}
                        onChange={(e) => {
                            const id = e.target.value;
                            const name =
                                notionDBs.find((db) => db.id === settings.notion_db_id)
                                    ?.properties?.[id]?.name ?? "Unknown";

                            setSettings((prev) => ({
                                ...prev,
                                date_property_id: id,
                                date_property_name: name,
                            }));
                        }}
                    >
                        <option value="" disabled>Select date property</option>
                        {Object.entries(
                            notionDBs.find((db) => db.id === settings.notion_db_id)?.properties ?? {}
                        )
                            .filter(([_, prop]) => prop.type === "date")
                            .map(([id, prop]) => (
                                <option key={id} value={id}>
                                    {prop.name}
                                </option>
                            ))}
                    </select>
                ) : (
                    <span>{settings.date_property_name || "(No date selected)"}</span>
                )}
            </div>

            <div>
                <label>Connected to OpenAI:</label>
                <span>{settings.has_openai_key ? "✅" : "❌"}</span>
            </div>

            {!dateValid && (
                <p className="settings-warning">
                    Please select a date property before saving.
                </p>
            )}

            <button
                onClick={saveSettings}
                disabled={!dateValid}
                className={`settings-button ${dateValid ? "enabled" : "disabled"}`}
            >
                Save Settings
            </button>

            {status && <p className="settings-status">{status}</p>}
        </div>
    );
}
