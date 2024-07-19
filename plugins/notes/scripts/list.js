const notes = Application("Notes");

/* // Iterate through each list and get reminders
folders.forEach(function (folder) {
	console.log("List: " + folder.name());
	const reminders = folder.notes();
	console.log("Reminders: " + reminders.length);
}); */

// Note : you can't do for const name of method() directly
// You need to first store the method() in a variable and then iterate over it
const accounts = notes.accounts();
for (const account of accounts) {
	const accountName = account.name();
	const folders = notes.folders();
	for (const folder of folders) {
		const notes = folder.notes();

		const folderName = folder.name();
		for (const note of notes) {
			console.log(
				JSON.stringify({
					id: note.id(),
					name: note.name(),
					creationDate: note.creationDate(),
					modificationDate: note.modificationDate(),
					htmlBody: note.body(),
					folder: folderName,
					account: accountName,
				}),
			);
		}
	}
}
