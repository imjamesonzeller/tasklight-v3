import { Routes, Route } from "react-router-dom"
import Input from "./Input"
import Settings from "./Settings"

export default function App() {
    return (
        <Routes>
            <Route path="/" element={<Input />} />
            <Route path="/settings" element={<Settings />} />
        </Routes>
    )
}