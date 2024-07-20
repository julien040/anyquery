const safari = Application("Safari");

const windows = safari.windows();

for (const window of windows) {
    if (window.index() === %d) {
        const tabs = window.tabs.whose({ index: %d });
        tabs[0].url = "%s";
    }
}
