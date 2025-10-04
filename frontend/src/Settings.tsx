import {useEffect, useMemo, useState} from "react"
import {SettingsService as s} from "../bindings/github.com/imjamesonzeller/tasklight-v3/settingsservice"
import {DatabaseMinimal, NotionService as n,} from "../bindings/github.com/imjamesonzeller/tasklight-v3"
import "../public/settings.css"
import {Events} from "@wailsio/runtime"
import {PauseHotkey, ResumeHotkey} from "../bindings/github.com/imjamesonzeller/tasklight-v3/hotkeyservice.ts"

type SelectNotionDBProps = {
    databases: DatabaseMinimal[]
    value: string
    onChange: (value: string) => void
    className?: string
    disabled?: boolean
}

function SelectNotionDB({ databases, value, onChange, className, disabled }: SelectNotionDBProps) {
    const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
        onChange(event.target.value)
    }

    const selectClasses = ["input-control", "select-control", className]
        .filter(Boolean)
        .join(" ")

    const isDisabled = disabled || databases.length === 0

    return (
        <div className={`select-wrapper${isDisabled ? " select-wrapper--disabled" : ""}`}>
            <select
                value={value}
                onChange={handleChange}
                className={selectClasses}
                disabled={isDisabled}
            >
                <option value="" disabled>
                    {databases.length === 0 ? "No databases available" : "Select a database"}
                </option>
                {databases.map((db) => (
                    <option key={db.id} value={db.id}>
                        {db.title?.[0]?.text?.content || `Untitled (${db.id.slice(0, 6)}…)`}
                    </option>
                ))}
            </select>
        </div>
    )
}

const tabs = [
    {
        id: "general",
        label: "General",
        description: "Theme, launch behaviour, and overall preferences.",
    },
    {
        id: "shortcuts",
        label: "Shortcuts",
        description: "Configure Tasklight's global hotkeys.",
    },
    {
        id: "ai",
        label: "AI",
        description: "Bring your own OpenAI key and manage usage.",
    },
    {
        id: "notion",
        label: "Notion",
        description: "Connect databases and pick date fields for syncing.",
    },
] as const

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
    })

    const [status, setStatus] = useState("")
    const [notionDBs, setNotionDBs] = useState<DatabaseMinimal[]>([])
    const [hasMultipleDateProps, setHasMultipleDateProps] = useState(false)
    const [dateValid, setDateValid] = useState(true)
    const [recordingHotkey, setRecordingHotkey] = useState(false)
    const [openAIKey, setOpenAIKey] = useState("")
    const [activeTab, setActiveTab] = useState<(typeof tabs)[number]["id"]>("general")

    useEffect(() => {
        document.body.dataset.theme = settings.theme === "dark" ? "dark" : "light"
        return () => {
            delete document.body.dataset.theme
        }
    }, [settings.theme])

    useEffect(() => {
        s.GetSettings()
            .then((res) => setSettings(res))
            .catch((err) => setStatus("❌ Failed to load settings: " + err.message))
    }, [])

    useEffect(() => {
        getNotionDBs()
    }, [])

    useEffect(() => {
        const selected = notionDBs.find((db) => db.id === settings.notion_db_id)
        if (!selected) return

        const dateProps = Object.entries(selected.properties ?? {}).filter(
            ([_, prop]) => prop.type === "date"
        )

        if (dateProps.length > 1) {
            setHasMultipleDateProps(true)
            setDateValid(settings.date_property_id !== "")
        } else {
            setHasMultipleDateProps(false)

            if (dateProps.length === 1) {
                const [id, prop] = dateProps[0]
                setSettings((prev) => ({
                    ...prev,
                    date_property_id: id,
                    date_property_name: prop.name,
                }))
                setDateValid(true)
            } else {
                setSettings((prev) => ({
                    ...prev,
                    date_property_id: "",
                    date_property_name: "",
                }))
                setDateValid(false)
            }
        }
    }, [settings.notion_db_id, notionDBs, settings.date_property_id])

    useEffect(() => {
        const off = Events.On("Backend:NotionAccessToken", async (ev) => {
            const success = ev.data as boolean
            if (success) {
                try {
                    const updatedSettings = await s.GetSettings()
                    setSettings(updatedSettings)

                    const dbResponse = await n.GetNotionDatabases()
                    setNotionDBs(dbResponse?.results ?? [])

                    setStatus("✅ Notion connected and databases refreshed.")
                } catch (err: any) {
                    setStatus(
                        "❌ Notion connected, but failed to refresh: " + (err.message ?? String(err))
                    )
                }
            } else {
                setStatus("❌ Failed to connect Notion.")
            }
        })

        return () => {
            off()
        }
    }, [])

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        const { name, value, type } = e.target

        setSettings((prev) => ({
            ...prev,
            [name]:
                type === "checkbox"
                    ? (e.target as HTMLInputElement).checked
                    : value,
        }))
    }

    const handleOpenAIChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setOpenAIKey(e.target.value)
    }

    const saveSettings = async () => {
        if (!dateValid) {
            setStatus("⚠️ Select a date property before saving.")
            setActiveTab("notion")
            return
        }

        try {
            if (settings.use_open_ai) {
                const trimmedKey = openAIKey.trim()
                if (!settings.has_openai_key && trimmedKey === "") {
                    setStatus("❌ Enter your OpenAI API key to enable BYOK.")
                    setActiveTab("ai")
                    return
                }

                if (trimmedKey !== "") {
                    await s.SaveOpenAIKey(trimmedKey)
                    setSettings((prev) => ({
                        ...prev,
                        has_openai_key: true,
                    }))
                    setOpenAIKey("")
                }
            }

            await s.UpdateSettingsFromFrontend(settings)
            setStatus("✅ Preferences saved.")
        } catch (err: any) {
            setStatus("❌ Failed to save settings: " + (err.message ?? String(err)))
        }
    }

    const connectNotion = async () => {
        try {
            setStatus("🔄 Opening Notion authorization…")
            await n.StartOAuth()
        } catch (err: any) {
            setStatus("❌ Failed to connect Notion: " + (err.message ?? String(err)))
        }
    }

    const getNotionDBs = async () => {
        try {
            const res = await n.GetNotionDatabases()
            const results = res?.results ?? []

            setNotionDBs(results)

            const selected = results.find((db) => db.id === settings.notion_db_id)
            if (selected?.has_multiple_date_props) {
                setHasMultipleDateProps(true)
            } else {
                setHasMultipleDateProps(false)
            }
        } catch (err: any) {
            const message = err?.message ?? String(err)
            if (message && !message.includes("401")) {
                setStatus("⚠️ Unable to load Notion databases: " + message)
            }
            setNotionDBs([])
        }
    }

    const resetOpenAI = async () => {
        try {
            await s.ClearOpenAIKey()
            setOpenAIKey("")
            setSettings((prev) => ({
                ...prev,
                has_openai_key: false,
                use_open_ai: false,
            }))
            setStatus("✅ OpenAI key cleared from Keychain.")
        } catch (err: any) {
            setStatus("❌ Failed to clear OpenAI key: " + (err.message ?? String(err)))
        }
    }

    const startRecordingHotkey = async () => {
        setRecordingHotkey(true)
        setStatus("⌨️ Waiting for hotkey…")

        await PauseHotkey()

        const pressedKeys = new Set<string>()
        const modifiers = new Set<string>()

        const keyMap: Record<string, string> = {
            Control: "ctrl",
            Meta: "cmd",
            Alt: "option",
            Shift: "shift",
        }

        const downHandler = (e: KeyboardEvent) => {
            e.preventDefault()

            if (e.ctrlKey) modifiers.add("ctrl")
            if (e.metaKey) modifiers.add("cmd")
            if (e.altKey) modifiers.add("option")
            if (e.shiftKey) modifiers.add("shift")

            const key = e.key

            if (!keyMap[key] && key !== " ") {
                pressedKeys.add(key.toLowerCase())
            } else if (key === " ") {
                pressedKeys.add("space")
            }
        }

        const upHandler = async (_e: KeyboardEvent) => {
            if (pressedKeys.size === 0) return

            const combo = [...modifiers, ...pressedKeys].join("+")

            setSettings((prev) => ({ ...prev, hotkey: combo }))
            setStatus(`✅ Hotkey set to ${combo}`)
            setRecordingHotkey(false)

            window.removeEventListener("keydown", downHandler)
            window.removeEventListener("keyup", upHandler)

            await ResumeHotkey()
        }

        window.addEventListener("keydown", downHandler)
        window.addEventListener("keyup", upHandler)
    }

    const notionConnected = settings.has_notion_secret
    const datePropertyLabel = settings.date_property_name || "(No date selected)"

    const statusTone = useMemo(() => {
        if (!status) return ""
        if (status.startsWith("✅")) return "positive"
        if (status.startsWith("❌")) return "negative"
        if (status.startsWith("⚠️")) return "warning"
        if (status.startsWith("🔄")) return "neutral"
        if (status.startsWith("⌨️")) return "neutral"
        return "info"
    }, [status])

    const renderGeneral = () => (
        <>
            <section className="settings-card">
                <header className="settings-card-header">
                    <h2>Appearance</h2>
                    <p>Choose how Tasklight looks when it pops into view.</p>
                </header>
                <div className="settings-field">
                    <label className="field-label">Theme</label>
                    <div className="select-wrapper">
                        <select
                            name="theme"
                            value={settings.theme}
                            onChange={handleChange}
                            className="input-control select-control"
                        >
                            <option value="light">Light</option>
                            <option value="dark">Dark</option>
                        </select>
                    </div>
                </div>
            </section>

            <section className="settings-card">
                <header className="settings-card-header">
                    <h2>Launch & Behaviour</h2>
                    <p>Keep Tasklight ready without showing a dock icon.</p>
                </header>
                <label className="toggle">
                    <input
                        type="checkbox"
                        name="launch_on_startup"
                        checked={settings.launch_on_startup}
                        onChange={handleChange}
                    />
                    <span className="toggle-track">
                        <span className="toggle-thumb" />
                    </span>
                    <div className="toggle-copy">
                        <span>Launch Tasklight on login</span>
                        <p>Your capture window is ready right after reboot.</p>
                    </div>
                </label>
            </section>
        </>
    )

    const renderShortcuts = () => (
        <section className="settings-card">
            <header className="settings-card-header">
                <h2>Global Hotkey</h2>
                <p>Summon Tasklight instantly from anywhere.</p>
            </header>
            <div className="hotkey-row">
                <input
                    id="hotkeyInput"
                    type="text"
                    name="hotkey"
                    value={settings.hotkey}
                    disabled
                    readOnly
                    className="input-control input-control--readonly"
                />
                <button
                    type="button"
                    onClick={startRecordingHotkey}
                    className={`btn btn-secondary ${recordingHotkey ? "btn-recording" : ""}`}
                >
                    {recordingHotkey ? "Press keys…" : "Change"}
                </button>
            </div>
        </section>
    )

    const renderAI = () => (
        <section className="settings-card">
            <header className="settings-card-header">
                <h2>Bring Your Own Key</h2>
                <p>Run parsing through your OpenAI account, stored securely in Keychain.</p>
            </header>

            <label className="toggle">
                <input
                    type="checkbox"
                    name="use_open_ai"
                    checked={settings.use_open_ai}
                    onChange={handleChange}
                />
                <span className="toggle-track">
                    <span className="toggle-thumb" />
                </span>
                <div className="toggle-copy">
                    <span>Use your own OpenAI API key</span>
                    <p>Toggle off to fall back to Tasklight's managed key when available.</p>
                </div>
            </label>

            {settings.use_open_ai && (
                <div className="settings-field">
                    <label className="field-label">OpenAI API key</label>
                    <p className="field-helper">Tasklight only stores this encrypted in your macOS Keychain.</p>
                    <div className="hotkey-row">
                        <input
                            type="password"
                            name="openai_api_key"
                            id="openaikey"
                            value={openAIKey}
                            onChange={handleOpenAIChange}
                            placeholder={
                                settings.has_openai_key
                                    ? "Stored securely in Keychain"
                                    : "sk-live-..."
                            }
                            className="input-control"
                        />
                        <button
                            type="button"
                            onClick={resetOpenAI}
                            className="btn btn-ghost"
                        >
                            Remove
                        </button>
                    </div>
                </div>
            )}
        </section>
    )

    const renderNotion = () => (
        <>
            <section className="settings-card">
                <header className="settings-card-header">
                    <h2>Notion Connection</h2>
                    <p>Authorise Tasklight and pick the database that receives new tasks.</p>
                </header>
                <div className="notion-connection">
                    <span
                        className={`status-chip ${
                            notionConnected ? "status-chip--positive" : "status-chip--negative"
                        }`}
                    >
                        {notionConnected ? "Connected" : "Not connected"}
                    </span>
                    <button type="button" onClick={connectNotion} className="btn btn-primary">
                        {notionConnected ? "Manage connection" : "Connect to Notion"}
                    </button>
                </div>
            </section>

            <section className="settings-card">
                <header className="settings-card-header">
                    <h2>Database & Date Property</h2>
                    <p>Tell Tasklight where to store tasks and which date field to hydrate.</p>
                </header>
                <div className="settings-field">
                    <label className="field-label">Task database</label>
                    <SelectNotionDB
                        databases={notionDBs}
                        value={settings.notion_db_id}
                        onChange={(value) =>
                            setSettings((prev) => ({ ...prev, notion_db_id: value }))
                        }
                        disabled={!notionConnected}
                    />
                </div>
                <div className="settings-field">
                    <label className="field-label">Date property</label>
                    {hasMultipleDateProps ? (
                        <div className="select-wrapper">
                            <select
                                value={settings.date_property_id}
                                onChange={(e) => {
                                    const id = e.target.value
                                    const name =
                                        notionDBs.find((db) => db.id === settings.notion_db_id)?.properties?.[id]
                                            ?.name ?? "Unknown"

                                    setSettings((prev) => ({
                                        ...prev,
                                        date_property_id: id,
                                        date_property_name: name,
                                    }))
                                }}
                                className="input-control select-control"
                            >
                                <option value="" disabled>
                                    Select date property
                                </option>
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
                        </div>
                    ) : (
                        <div className="status-chip status-chip--neutral">
                            {datePropertyLabel}
                        </div>
                    )}
                </div>

                {!dateValid && (
                    <p className="inline-warning">Select a date property before saving.</p>
                )}
            </section>
        </>
    )

    const renderContent = () => {
        switch (activeTab) {
            case "general":
                return renderGeneral()
            case "shortcuts":
                return renderShortcuts()
            case "ai":
                return renderAI()
            case "notion":
                return renderNotion()
            default:
                return null
        }
    }

    const themeClass = settings.theme === "dark" ? "settings-shell--dark" : "settings-shell--light"

    return (
        <div className={`settings-shell ${themeClass}`}>
            <div className="settings-scroll-region">
                <div className="settings-layout">
                    <aside className="settings-sidebar">
                        <div className="sidebar-header">
                            <h1>Tasklight Preferences</h1>
                            <p>Fine-tune Tasklight without ever showing the dock icon.</p>
                        </div>
                        <nav className="settings-nav">
                            {tabs.map((tab) => (
                                <button
                                    key={tab.id}
                                    type="button"
                                    className={`settings-nav-button ${
                                        activeTab === tab.id ? "settings-nav-button--active" : ""
                                    }`}
                                    onClick={() => setActiveTab(tab.id)}
                                >
                                    <span className="settings-nav-label">{tab.label}</span>
                                    <span className="settings-nav-desc">{tab.description}</span>
                                </button>
                            ))}
                        </nav>
                    </aside>

                    <main className="settings-content">
                        {renderContent()}

                        <footer className="settings-footer">
                            <button
                                type="button"
                                onClick={saveSettings}
                                disabled={!dateValid}
                                className="btn btn-accent"
                            >
                                Save preferences
                            </button>

                            {status && (
                                <div className={`status-banner status-banner--${statusTone || "info"}`}>
                                    {status}
                                </div>
                            )}
                        </footer>
                    </main>
                </div>
            </div>
        </div>
    )
}
