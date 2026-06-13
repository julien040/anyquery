on run argv
	tell application "Safari"
		if not (exists document 1) then reopen
		make new tab with properties {URL:(item 1 of argv)} at end of tabs of window %d
	end tell
end run
