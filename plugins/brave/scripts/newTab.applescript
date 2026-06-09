on run argv
	tell application "Brave"
		make new tab with properties {URL:(item 1 of argv)} at end of tabs of first window
	end tell
end run
