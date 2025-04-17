export default function Settings() {
    return (
        <div className="p-4 flex flex-col gap-4">
            <h2 className="text-xl font-bold">Settings</h2>

            <div>
                <label className="block">Theme</label>
                <select>
                    <option value="light">Light</option>
                    <option value="dark">Dark</option>
                    <option value="system">System</option>
                </select>
            </div>

            <div>
                <label className="block">Use OpenAI</label>
                <input type="checkbox" />
            </div>

            <div>
                <label className="block">OpenAI API Key</label>
                <input type="text" className="w-full" />
            </div>

            <div>
                <label className="block">Notion API Key</label>
                <input type="text" className="w-full" />
            </div>

            <div>
                <label className="block">Notion Database ID</label>
                <input type="text" className="w-full" />
            </div>

            <div>
                <label className="block">Date Property Name</label>
                <input type="text" className="w-full" />
            </div>

            <button className="mt-4 px-4 py-2 bg-blue-500 text-white rounded">
                Save
            </button>
        </div>
    )
}