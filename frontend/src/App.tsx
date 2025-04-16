import { useEffect, useRef, useState, KeyboardEvent, ChangeEvent } from "react";
// import "./App.css";
import { WindowService as ws, TaskService as ts } from "../bindings/github.com/imjamesonzeller/tasklight-v3"
import { Events } from '@wailsio/runtime';
// @ts-ignore
import { WailsEvent } from "@wailsio/runtime/types/events";

function App() {
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
      ws.ToggleVisibility(window);
      setName("");
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

    Events.On("wails:focus", focusInput);
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
      <div className="spotlight-container">
        <div className="spotlight-box">
          <input
              ref={inputRef}
              className="spotlight-input"
              type="text"
              placeholder="Type your task..."
              value={name}
              onKeyDown={handleKeyDown}
              onChange={updateName}
              autoComplete="off"
          />
        </div>

        {resultText && <div className="spotlight-results">{resultText}</div>}
      </div>
  );
}

export default App;