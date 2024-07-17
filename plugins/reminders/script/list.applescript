(* 
The MIT License (MIT)

Copyright (c) 2014 Alex Morega

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*)
on encode(value)
	set type to class of value
	if type = integer or type = boolean
		return value as text
	else if type = text
		return encodeString(value)
	else if type = list
		return encodeList(value)
	else if type = script
		return value's toJson()
	else
		error "Unknown type " & type
	end
end


on encodeList(value_list)
	set out_list to {}
	repeat with value in value_list
		copy encode(value) to end of out_list
	end
	return "[" & join(out_list, ", ") & "]"
end


on encodeString(value)
    (* Check if missing value *)
    if value is missing value then
        return "null"
    end if


	set rv to ""
	set codepoints to id of value

	if (class of codepoints) is not list
		set codepoints to {codepoints}
	end

	repeat with codepoint in codepoints
		set codepoint to codepoint as integer
		if codepoint = 34
			set quoted_ch to "\\\""
		else if codepoint = 92 then
			set quoted_ch to "\\\\"
		else if codepoint >= 32 and codepoint < 127
			set quoted_ch to character id codepoint
		else
			set quoted_ch to "\\u" & hex4(codepoint)
		end
		set rv to rv & quoted_ch
	end
	return "\"" & rv & "\""
end


on join(value_list, delimiter)
	set original_delimiter to AppleScript's text item delimiters
	set AppleScript's text item delimiters to delimiter
	set rv to value_list as text
	set AppleScript's text item delimiters to original_delimiter
	return rv
end


on hex4(n)
	set digit_list to "0123456789abcdef"
	set rv to ""
	repeat until length of rv = 4
		set digit to (n mod 16)
		set n to (n - digit) / 16 as integer
		set rv to (character (1+digit) of digit_list) & rv
	end
	return rv
end


on createDictWith(item_pairs)
	set item_list to {}

	script Dict
		on setkv(key, value)
			copy {key, value} to end of item_list
		end

		on toJson()
			set item_strings to {}
			repeat with kv in item_list
				set key_str to encodeString(item 1 of kv)
				set value_str to encode(item 2 of kv)
				copy key_str & ": " & value_str to end of item_strings
			end
			return "{" & join(item_strings, ", ") & "}"
		end
	end

	repeat with pair in item_pairs
		Dict's setkv(item 1 of pair, item 2 of pair)
	end

	return Dict
end


on createDict()
	return createDictWith({})
end

(* 
Copyright (c) 2024 Julien CAGNIART
for lines below this comment
 *)

(* tell application "Reminders"
	repeat with aList in every list of first account
        set aListName to name of aList
		if aListName = "Devoirs" then
            make reminder at end of reminders of aList with properties {name:"Faire les devoirs", due date:(current date) + (1 * days)}
            exit repeat
        end if
	end repeat
end tell *)

on encodeDate (aDate)
    if aDate is missing value then
        return "null"
    end if
    set yearD to year of aDate
    set monthD to month of aDate
    set monthInteger to monthD as integer
    set dayD to day of aDate
    if monthInteger is less than 10 then
        set formattedMonth to "0" & monthInteger
    else
        set formattedMonth to monthInteger as string
    end if
    if dayD is less than 10 then
        set formattedDay to "0" & dayD
    else
        set formattedDay to dayD as string
    end if
    set allStr to yearD & "-" & formattedMonth & "-" & formattedDay
    return "\"" & allStr & "\""
end encodeDate

tell application "Reminders"
repeat with aList in every list of first account
    set aListName to name of aList
    repeat with itemA in every reminder of aList
        set idA to id of itemA
        set nameA to name of itemA
        set bodyA to body of itemA
        set completedA to completed of itemA
        set dueDateA to due date of itemA
        set priorityA to priority of itemA
        log "{\"id\":" & my encodeString(idA)& ",\"name\":" & my encodeString(nameA) & ",\"body\":" & my encodeString(bodyA) & ",\"completed\":" & completedA & ",\"due_date\":" & my encodeDate(dueDateA) & ",\"priority\":" & priorityA & ",\"list\":" & my encodeString(aListName) & "}"
    end repeat
    end repeat
end tell

(* -- Define the variables
set dayVar to 16
set monthVar to 7
set yearVar to 2024
set hourVar to 14
set minuteVar to 30

-- Create a date object
set newDate to current date

-- Set the date components
set day of newDate to dayVar
set month of newDate to monthVar
set year of newDate to yearVar
set hours of newDate to hourVar
set minutes of newDate to minuteVar

-- Display the new date
return newDate *)