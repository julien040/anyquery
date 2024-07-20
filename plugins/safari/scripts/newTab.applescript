tell application "Safari"
	if not (exists document 1) then reopen
	make new tab with properties {URL:"%s"} at end of tabs of window %d
	
end tell