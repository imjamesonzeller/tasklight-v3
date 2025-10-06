import {useCallback, useEffect, useMemo, useRef, useState} from "react"
import {SettingsService as s} from "../bindings/github.com/imjamesonzeller/tasklight-v3/settingsservice"
import {DatabaseMinimal, NotionService as n,} from "../bindings/github.com/imjamesonzeller/tasklight-v3"
import "../public/settings.css"
import {Events, Browser} from "@wailsio/runtime"
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

type HelpView = "root" | "about" | "acknowledgements" | "resetConfirm"

type Acknowledgement = {
    name: string
    description: string
    url: string
}

const acknowledgements: Acknowledgement[] = [
    {
        name: "Wails",
        description: "Desktop runtime marrying Go backends with modern web UIs.",
        url: "https://wails.io",
    },
    {
        name: "React",
        description: "Component model powering the interactive settings surface.",
        url: "https://react.dev",
    },
    {
        name: "Notion API",
        description: "Official API layer Tasklight relies on for workspace sync.",
        url: "https://developers.notion.com",
    },
    {
        name: "go-autostart",
        description: "Cross-platform helpers for managing login launch agents.",
        url: "https://github.com/protonmail/go-autostart",
    },
]

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
    const [notionConnecting, setNotionConnecting] = useState(false)
    const [notionDBs, setNotionDBs] = useState<DatabaseMinimal[]>([])
    const [hasMultipleDateProps, setHasMultipleDateProps] = useState(false)
    const [dateValid, setDateValid] = useState(true)
    const [recordingHotkey, setRecordingHotkey] = useState(false)
    const [openAIKey, setOpenAIKey] = useState("")
    const [activeTab, setActiveTab] = useState<(typeof tabs)[number]["id"]>("general")
    const notionConnectTimeoutRef = useRef<number | null>(null)
    const [helpOpen, setHelpOpen] = useState(false)
    const [helpView, setHelpView] = useState<HelpView>("root")
    const [helpSelectionIndex, setHelpSelectionIndex] = useState(0)
    const [appVersion, setAppVersion] = useState("")
    const [helpError, setHelpError] = useState<string | null>(null)
    const [clearingCache, setClearingCache] = useState(false)
    const helpModalRef = useRef<HTMLDivElement | null>(null)
    const helpMenuItemRefs = useRef<Array<HTMLButtonElement | null>>([])
    const helpLauncherRef = useRef<HTMLButtonElement | null>(null)
    const confirmButtonRef = useRef<HTMLButtonElement | null>(null)
    const previousFocusRef = useRef<HTMLElement | null>(null)

    const clearNotionConnectTimeout = useCallback(() => {
        if (notionConnectTimeoutRef.current !== null) {
            window.clearTimeout(notionConnectTimeoutRef.current)
            notionConnectTimeoutRef.current = null
        }
    }, [])

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
        if (!helpOpen || appVersion) {
            return
        }

        s.GetAppVersion()
            .then((version) => setAppVersion(version))
            .catch(() => setAppVersion("development"))
    }, [helpOpen, appVersion])

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
            clearNotionConnectTimeout()
            setNotionConnecting(false)

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
            clearNotionConnectTimeout()
        }
    }, [clearNotionConnectTimeout])

    useEffect(() => {
        if (!helpOpen) {
            return
        }

        const modal = helpModalRef.current
        if (!modal) {
            return
        }

        previousFocusRef.current = document.activeElement as HTMLElement | null

        const focusInitialElement = () => {
            if (helpView === "root") {
                const items = helpMenuItemRefs.current.filter((btn): btn is HTMLButtonElement => Boolean(btn))
                if (items.length === 0) {
                    return
                }

                const targetIndex = items[helpSelectionIndex] ? helpSelectionIndex : 0
                if (!items[targetIndex]) {
                    setHelpSelectionIndex(0)
                    items[0].focus()
                    return
                }
                items[targetIndex].focus()
                return
            }

            if (helpView === "resetConfirm" && confirmButtonRef.current) {
                confirmButtonRef.current.focus()
                return
            }

            const defaultFocus = modal.querySelector<HTMLElement>("[data-default-focus='true']")
            if (defaultFocus) {
                defaultFocus.focus()
                return
            }

            const fallback = modal.querySelector<HTMLElement>(
                "button:not([disabled]), [href], [tabindex]:not([tabindex='-1'])"
            )
            fallback?.focus()
        }

        focusInitialElement()

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                event.preventDefault()
                if (helpView !== "root") {
                    setHelpView("root")
                    setHelpSelectionIndex(0)
                } else {
                    setHelpOpen(false)
                }
                return
            }

            if (event.key === "Tab") {
                const focusable = Array.from(
                    modal.querySelectorAll<HTMLElement>(
                        "button:not([disabled]), [href], [tabindex]:not([tabindex='-1'])"
                    )
                ).filter((el) => !el.hasAttribute("aria-hidden"))

                if (focusable.length === 0) {
                    return
                }

                const first = focusable[0]
                const last = focusable[focusable.length - 1]
                if (!event.shiftKey && document.activeElement === last) {
                    event.preventDefault()
                    first.focus()
                } else if (event.shiftKey && document.activeElement === first) {
                    event.preventDefault()
                    last.focus()
                }
                return
            }

            if (helpView === "root" && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
                const items = helpMenuItemRefs.current.filter((btn): btn is HTMLButtonElement => Boolean(btn))
                if (!items.length) {
                    return
                }
                event.preventDefault()
                const direction = event.key === "ArrowDown" ? 1 : -1
                let nextIndex = helpSelectionIndex + direction
                if (nextIndex < 0) {
                    nextIndex = items.length - 1
                }
                if (nextIndex >= items.length) {
                    nextIndex = 0
                }
                setHelpSelectionIndex(nextIndex)
                items[nextIndex].focus()
            }
        }

        document.addEventListener("keydown", handleKeyDown)
        return () => {
            document.removeEventListener("keydown", handleKeyDown)
            const previouslyFocused = previousFocusRef.current
            if (previouslyFocused) {
                previouslyFocused.focus()
            }
        }
    }, [helpOpen, helpView, helpSelectionIndex])

    useEffect(() => {
        if (!helpOpen) {
            setHelpView("root")
            setHelpSelectionIndex(0)
            setHelpError(null)
        }
    }, [helpOpen])

    useEffect(() => {
        setHelpError(null)
    }, [helpView])

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
        if (notionConnecting) {
            return
        }

        try {
            setNotionConnecting(true)
            clearNotionConnectTimeout()
            notionConnectTimeoutRef.current = window.setTimeout(() => {
                setNotionConnecting(false)
                setStatus("⚠️ Notion authorization timed out. Try again.")
            }, 120000)
            setStatus("🔄 Opening Notion authorization…")
            await n.StartOAuth()
        } catch (err: any) {
            clearNotionConnectTimeout()
            setNotionConnecting(false)
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

    const openExternal = useCallback((url: string) => {
        Browser.OpenURL(url)
    }, [])
s
    const helpItems = useMemo(
        () => [
            {
                id: "about",
                label: "About / Credits",
                description: "See who built Tasklight and which version you're on.",
                action: () => setHelpView("about"),
            },
            {
                id: "bug",
                label: "Report a Bug",
                description: "Found an issue? Send it to the Tasklight tracker.",
                action: () => openExternal("https://jamesonzeller.com/tasklight/bugs"),
            },
            {
                id: "feature",
                label: "Request a Feature",
                description: "Share the workflow improvements you want next.",
                action: () => openExternal("https://jamesonzeller.com/tasklight/feature-request"),
            },
            {
                id: "contact",
                label: "Contact / Feedback",
                description: "Drop Jameson a note directly from your mail client.",
                action: () => {
                    window.location.href = "mailto:hello@jamesonzeller.com"
                },
            },
            {
                id: "privacy",
                label: "Privacy & Data",
                description: "Review how Tasklight handles data and telemetry.",
                action: () => openExternal("https://jamesonzeller.com/tasklight/privacy"),
            },
            {
                id: "reset",
                label: "Reset / Clear Cache",
                description: "Erase cached data and require a fresh sign-in.",
                action: () => setHelpView("resetConfirm"),
            },
            {
                id: "oss",
                label: "Open Source / Acknowledgements",
                description: "Browse the core libraries that make Tasklight possible.",
                action: () => setHelpView("acknowledgements"),
            },
        ],
        [openExternal, setHelpView]
    )

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

    const handleClearCache = async () => {
        setHelpError(null)
        setClearingCache(true)
        try {
            const cleared = await s.ClearLocalCache()
            if (!cleared) {
                setHelpError("Cache did not clear. Please try again.")
                return
            }

            const refreshedSettings = await s.GetSettings()
            setSettings(refreshedSettings)
            setNotionDBs([])
            setHasMultipleDateProps(false)
            setDateValid(true)
            setNotionConnecting(false)
            clearNotionConnectTimeout()

            setStatus("✅ Local cache cleared. Sign in again to reconnect Notion.")
            setHelpOpen(false)
        } catch (err: any) {
            setHelpError("Unable to clear cache: " + (err?.message ?? String(err)))
        } finally {
            setClearingCache(false)
        }
    }

    const helpBackToRoot = () => {
        setHelpView("root")
        setHelpSelectionIndex(0)
    }

    const renderHelpContent = () => {
        switch (helpView) {
            case "about":
                return (
                    <div className="help-modal-content">
                        <header className="help-modal-header">
                            <button
                                type="button"
                                className="help-back"
                                onClick={helpBackToRoot}
                                data-default-focus="true"
                            >
                                ← Back
                            </button>
                            <h2 id="help-modal-title">About Tasklight</h2>
                            <button
                                type="button"
                                className="help-close"
                                onClick={() => setHelpOpen(false)}
                                aria-label="Close help"
                            >
                                ✕
                            </button>
                        </header>
                        <div className="help-about">
                            <p>Created by Jameson Zeller</p>
                            <p>Version: {appVersion || "…"}</p>
                            <a
                                href="https://jamesonzeller.com/tasklight"
                                target="_blank"
                                rel="noreferrer noopener"
                                className="help-link"
                            >
                                Visit the Tasklight site
                            </a>
                        </div>
                    </div>
                )
            case "acknowledgements":
                return (
                    <div className="help-modal-content">
                        <header className="help-modal-header">
                            <button
                                type="button"
                                className="help-back"
                                onClick={helpBackToRoot}
                                data-default-focus="true"
                            >
                                ← Back
                            </button>
                            <h2 id="help-modal-title">Open Source Thanks</h2>
                            <button
                                type="button"
                                className="help-close"
                                onClick={() => setHelpOpen(false)}
                                aria-label="Close help"
                            >
                                ✕
                            </button>
                        </header>
                        <div className="help-modal-scroll">
                            <div className="help-oss-list" role="list">
                                {acknowledgements.map((entry) => (
                                    <a
                                        key={entry.name}
                                        href={entry.url}
                                        target="_blank"
                                        rel="noreferrer noopener"
                                        className="help-oss-item"
                                        role="listitem"
                                    >
                                        <span className="help-oss-name">{entry.name}</span>
                                        <span className="help-oss-description">{entry.description}</span>
                                    </a>
                                ))}
                            </div>
                        </div>
                    </div>
                )
            case "resetConfirm":
                return (
                    <div className="help-modal-content">
                        <header className="help-modal-header">
                            <button
                                type="button"
                                className="help-back"
                                onClick={helpBackToRoot}
                                data-default-focus="true"
                            >
                                ← Back
                            </button>
                            <h2 id="help-modal-title">Reset Local Cache</h2>
                            <button
                                type="button"
                                className="help-close"
                                onClick={() => setHelpOpen(false)}
                                aria-label="Close help"
                            >
                                ✕
                            </button>
                        </header>
                        <div className="help-confirm">
                            <p>This will clear local cache and require sign-in again. Continue?</p>
                            {helpError && <p className="help-error" role="alert">{helpError}</p>}
                            <div className="help-confirm-actions">
                                <button
                                    type="button"
                                    className="btn btn-ghost"
                                    onClick={helpBackToRoot}
                                >
                                    Cancel
                                </button>
                                <button
                                    type="button"
                                    className="btn btn-critical"
                                    onClick={handleClearCache}
                                    disabled={clearingCache}
                                    ref={confirmButtonRef}
                                >
                                    {clearingCache ? "Clearing…" : "Yes, clear cache"}
                                </button>
                            </div>
                        </div>
                    </div>
                )
            default: {
                helpMenuItemRefs.current = []
                return (
                    <div className="help-modal-content">
                        <header className="help-modal-header">
                            <h2 id="help-modal-title">Need a hand?</h2>
                            <button
                                type="button"
                                className="help-close"
                                onClick={() => setHelpOpen(false)}
                                aria-label="Close help"
                                data-default-focus="true"
                            >
                                ✕
                            </button>
                        </header>
                        <p className="help-modal-subtitle">
                            Quick access to support, feedback, and maintenance tools.
                        </p>
                        <div className="help-modal-scroll">
                            <ul className="help-menu" role="menu">
                                {helpItems.map((item, index) => (
                                    <li key={item.id} role="none">
                                        <button
                                            type="button"
                                            role="menuitem"
                                            className="help-menu-item"
                                            onClick={() => item.action()}
                                            onFocus={() => setHelpSelectionIndex(index)}
                                            ref={(el) => {
                                                helpMenuItemRefs.current[index] = el
                                            }}
                                        >
                                            <span className="help-menu-label">{item.label}</span>
                                            <span className="help-menu-description">{item.description}</span>
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        </div>
                    </div>
                )
            }
        }
    }

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
                    <button
                        type="button"
                        onClick={connectNotion}
                        className="btn btn-primary"
                        disabled={notionConnecting}
                    >
                        {notionConnecting
                            ? "Waiting for Notion…"
                            : notionConnected
                              ? "Manage connection"
                              : "Connect to Notion"}
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

                        <button
                            type="button"
                            ref={helpLauncherRef}
                            className="help-fab"
                            title="Help"
                            aria-haspopup="dialog"
                            aria-expanded={helpOpen}
                            onClick={() => {
                                setHelpOpen(true)
                                setHelpView("root")
                                setHelpSelectionIndex(0)
                            }}
                        >
                            ?
                        </button>

                        {helpOpen && (
                            <div
                                className="help-modal-overlay"
                                role="presentation"
                                onClick={() => setHelpOpen(false)}
                            >
                                <div
                                    className="help-modal-shell"
                                    role="dialog"
                                    aria-modal="true"
                                    aria-labelledby="help-modal-title"
                                    onClick={(event) => event.stopPropagation()}
                                    ref={helpModalRef}
                                >
                                    {renderHelpContent()}
                                </div>
                            </div>
                        )}
                    </main>
                </div>
            </div>
        </div>
    )
}
