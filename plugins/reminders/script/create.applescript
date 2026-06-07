on run argv
{{ if ne .Body "" }} set bodyR to item 2 of argv {{ end }}
set nameR to item 1 of argv
{{ if ne .Day "" }} set dayVar to {{.Day}} {{ end }}
{{ if ne .Month "" }} set monthVar to {{.Month}} {{ end }}
{{ if ne .Year "" }} set yearVar to {{.Year}} {{ end }}
{{ if ne .Hour "" }} set hourVar to {{.Hour}} {{ end }}
{{ if ne .Minute "" }} set minuteVar to {{.Minute}} {{ end }}
set ListName to item 3 of argv

set newDate to current date
{{if and (ne .Day "") (ne .Month "") (ne .Year "")}}
    set day of newDate to dayVar
    set month of newDate to monthVar
    set year of newDate to yearVar
{{ end }}
{{if and (ne .Minute "") (ne .Hour "")}}
    set hours of newDate to hourVar
    set minutes of newDate to minuteVar
{{ end }}

tell application "Reminders"
    make new reminder with properties {name:nameR{{ if ne .Body "" }}, body:bodyR {{end}}{{if and (ne .Day "") (ne .Month "") (ne .Year "")}}, due date:newDate{{end}}} at end of reminders of list ListName
end tell
end run
