const app = Application.currentApplication();
app.includeStandardAdditions = true;
const safari = Application("Safari");

const windows = safari.windows();

for (const window of windows) {
	const windowName = window.name();
	const windowIndex = window.index();
	const tabs = window.tabs();
	if (!tabs) {
		continue;
	}
	for (const tab of tabs) {
		console.log(
			JSON.stringify({
				title: tab.name(),
				url: tab.url(),
				windowName: windowName,
				windowIndex: windowIndex,
				index: tab.index(),
				visible: tab.visible(),
			}),
		);
	}
}
