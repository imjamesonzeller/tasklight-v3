import { useEffect, useRef, useState, KeyboardEvent, ChangeEvent } from "react";
// import "./App.css";
import { WindowService as ws, TaskService as ts } from "../bindings/github.com/imjamesonzeller/tasklight-v3"
import { Events } from '@wailsio/runtime';
// @ts-ignore
import { WailsEvent } from "@wailsio/runtime/types/events";

function Input() {
    const [resultText, setResultText] = useState<string>("");
    const [name, setName] = useState<string>("");
    const inputRef = useRef<HTMLInputElement>(null);
    const window: string = "main"

    const updateName = (e: ChangeEvent<HTMLInputElement>) => setName(e.target.value);

    const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
        if (e.key === "Enter") {
            processMessage();
        }

        if (e.key === "Escape") {
            e.preventDefault();
            ws.Hide(window);
            setName("");
        }

        if (e.metaKey && e.key === ",") {
            // Hotkey to open settings
            e.preventDefault()
            ws.Show("settings")
            ws.Hide(window)
        }
    };

    const processMessage = () => {
        if (!name.trim()) {
            setResultText("⚠️ Input cannot be empty.");
            return;
        }

        ts.ProcessMessage(name)
            .then(() => {
                setName("");
            })
            .catch(() => {
                setResultText("❌ An error occurred while processing the message.");
            });
    };

    useEffect(() => {
        const focusInput = () => {
            if (inputRef.current) {
                inputRef.current.focus();
                setResultText("");
            }
        };

        setTimeout(() => {
            if (document.hasFocus()) {
                focusInput();
            }
        }, 50);

        // Properly capture the 'off' function for cleanup
        const off = Events.On("wails:focus", focusInput);

        // CLEAN UP: remove listener on unmount
        return () => {
            off();
        };
    }, []);

    useEffect(() => {
        const off = Events.On("Backend:ErrorEvent", (ev: WailsEvent) => {
            setResultText(`❌ Error: ${ev.data}`);
        });

        return () => {
            off(); // <-- remove listener on unmount
        };
    }, []);

    return (
        <div className="spotlight-container undraggable">
            <div className="spotlight-box undraggable">
                <input
                    ref={inputRef}
                    className="spotlight-input undraggable"
                    type="text"
                    placeholder="Type your task..."
                    value={name}
                    onKeyDown={handleKeyDown}
                    onChange={updateName}
                    autoComplete="off"
                />
            </div>

            {resultText && <div className="spotlight-results undraggable">{resultText}</div>}
        </div>
    );
}

export default Input;