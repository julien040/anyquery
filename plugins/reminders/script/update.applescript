{{ if ne .Body "" }} set bodyR to "{{.Body}}" {{ end }}
{{ if ne .Name "" }} set nameR to "{{.Name}}" {{ end }}
{{ if ne .ID "" }} set idR to "{{.ID}}" {{ end }}
{{ if ne .Completed "" }} set completedR to {{.Completed}} {{ end }}
{{ if ne .Priority ""}} set priorityR to {{.Priority}} {{ end }}

{{ if ne .Day "" }} set dayVar to {{.Day}} {{ end }}
{{ if ne .Month "" }} set monthVar to {{.Month}} {{ end }}
{{ if ne .Year "" }} set yearVar to {{.Year}} {{ end }}
{{ if ne .Hour "" }} set hourVar to {{.Hour}} {{ end }}
{{ if ne .Minute "" }} set minuteVar to {{.Minute}} {{ end }}
{{ if and (ne .Day "") (ne .Month "") (ne .Year "") }}
set newDate to current date
set day of newDate to dayVar
set month of newDate to monthVar
set year of newDate to yearVar
{{ end }}
{{if and (ne .Minute "") (ne .Hour "")}}
set hours of newDate to hourVar
set minutes of newDate to minuteVar
{{ end }}


tell application "Reminders"
    set theReminder to reminder id idR
    {{if .Name}} set name of theReminder to nameR {{end}}
    {{if .Body}} set body of theReminder to bodyR {{end}}
    {{if .Completed}} set completed of theReminder to completedR {{end}}
    {{if .Priority}} set priority of theReminder to priorityR {{end}}
    {{if and (ne .Day "") (ne .Month "") (ne .Year "")}} set due date of theReminder to newDate {{end}}
end tell