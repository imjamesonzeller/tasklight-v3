# Notion Data Source Migration

Tasklight now targets Notion **data sources** (2025-09-03 API) when searching schemas and creating pages. Existing installs that previously stored only a database ID need to record a data source once after upgrading.

- Launch Tasklight and open Settings → Notion.
- Choose the data source where new tasks should be added. Tasklight auto-selects it when only one is available; otherwise pick the correct source from the list.
- Re-select the due-date property if prompted so it matches the data source schema.

After saving, the selected `notion_data_source_id` is persisted and reused for all future task creation. No additional migration steps are required.
