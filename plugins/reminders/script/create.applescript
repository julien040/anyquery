
{{ if ne .Body "" }} set bodyR to "{{.Body}}" {{ end }}
set nameR to "{{.Name}}"
{{ if ne .Day "" }} set dayVar to {{.Day}} {{ end }}
{{ if ne .Month "" }} set monthVar to {{.Month}} {{ end }}
{{ if ne .Year "" }} set yearVar to {{.Year}} {{ end }}
{{ if ne .Hour "" }} set hourVar to {{.Hour}} {{ end }}
{{ if ne .Minute "" }} set minuteVar to {{.Minute}} {{ end }}
set ListName to "{{.List}}"

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
    make new reminder with properties {name:nameR{{ if ne .Body "" }}, body:bodyR {{end}}{{if and (ne .Day "") (ne .Month "") (ne .Year "")}}, due date:newDate{{end}}} at end of reminders of list listName
end tell