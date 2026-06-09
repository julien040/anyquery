on run argv
set idR to item 4 of argv

tell application "Reminders"
    delete reminder id idR
end tell
end run
