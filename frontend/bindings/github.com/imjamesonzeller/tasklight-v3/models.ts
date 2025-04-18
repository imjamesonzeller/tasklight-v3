// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import {Create as $Create} from "@wailsio/runtime";

export class FrontendSettings {
    "notion_db_id": string;
    "use_open_ai": boolean;
    "theme": string;
    "launch_on_startup": boolean;
    "hotkey": string;
    "has_notion_secret": boolean;
    "has_openai_key": boolean;

    /** Creates a new FrontendSettings instance. */
    constructor($$source: Partial<FrontendSettings> = {}) {
        if (!("notion_db_id" in $$source)) {
            this["notion_db_id"] = "";
        }
        if (!("use_open_ai" in $$source)) {
            this["use_open_ai"] = false;
        }
        if (!("theme" in $$source)) {
            this["theme"] = "";
        }
        if (!("launch_on_startup" in $$source)) {
            this["launch_on_startup"] = false;
        }
        if (!("hotkey" in $$source)) {
            this["hotkey"] = "";
        }
        if (!("has_notion_secret" in $$source)) {
            this["has_notion_secret"] = false;
        }
        if (!("has_openai_key" in $$source)) {
            this["has_openai_key"] = false;
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new FrontendSettings instance from a string or object.
     */
    static createFrom($$source: any = {}): FrontendSettings {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new FrontendSettings($$parsedSource as Partial<FrontendSettings>);
    }
}

export class TaskInformation {
    "title": string;
    "date": string | null;

    /** Creates a new TaskInformation instance. */
    constructor($$source: Partial<TaskInformation> = {}) {
        if (!("title" in $$source)) {
            this["title"] = "";
        }
        if (!("date" in $$source)) {
            this["date"] = null;
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new TaskInformation instance from a string or object.
     */
    static createFrom($$source: any = {}): TaskInformation {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new TaskInformation($$parsedSource as Partial<TaskInformation>);
    }
}
