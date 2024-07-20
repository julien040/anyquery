const app = Application.currentApplication();
app.includeStandardAdditions = true;
const chromium = Application("Brave");

const windows = chromium.windows();

for (const window of windows) {
	const windowName = window.name();
	const windowID = window.id();
	const tabs = window.tabs();
	if (!tabs) {
		continue;
	}
	const activeTab = window.activeTab();
	const activeTabId = activeTab.id();

	for (const tab of tabs) {
		console.log(
			JSON.stringify({
				title: tab.name(),
				url: tab.url(),
				windowName: windowName,
				windowID: windowID,
				id: tab.id(),
				loading: tab.loading(),
				active: tab.id() === activeTabId,
			}),
		);
	}
}
