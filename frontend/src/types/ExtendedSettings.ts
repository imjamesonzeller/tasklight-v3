import type { FrontendSettings } from "../../bindings/github.com/imjamesonzeller/tasklight-v3";

// Extend the Wails-generated type with secrets
export interface ApplicationSettings extends FrontendSettings {
    notion_secret?: string;
    openai_api_key?: string;
}