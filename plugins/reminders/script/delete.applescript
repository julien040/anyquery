set idR to "{{.ID}}"

tell application "Reminders"
    delete reminder id idR
end tell