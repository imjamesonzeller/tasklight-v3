// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import {Call as $Call, Create as $Create} from "@wailsio/runtime";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import * as application$0 from "../../wailsapp/wails/v3/pkg/application/models.js";

export function SetApp(app: application$0.App | null): Promise<void> & { cancel(): void } {
    let $resultPromise = $Call.ByID(1245702848, app) as any;
    return $resultPromise;
}

export function StartHotkeyListener(): Promise<void> & { cancel(): void } {
    let $resultPromise = $Call.ByID(3811657959) as any;
    return $resultPromise;
}
