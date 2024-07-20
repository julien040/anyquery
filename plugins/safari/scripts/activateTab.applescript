tell application "Safari"
      if not (exists current tab of front window) then make new document -- if no window
      tell window %d
            set current tab to tab %d
      end tell
end tell