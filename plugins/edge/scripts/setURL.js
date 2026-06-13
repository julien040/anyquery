function run(argv) {
	const chromium = Application("Microsoft Edge");

	const windows = chromium.windows();

	for (const window of windows) {
		const tabs = window.tabs();
		for (const tab of tabs) {
			if (tab.id() === argv[0]) {
				tab.url = argv[1];
			}
		}
	}
}
