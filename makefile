FILES:= main.go

ifdef CC
CC := $(CC)
else
CC := "gcc -O2"
endif

TAGS := vtable fts5 sqlite_json sqlite_math_functions
 
BUILD_FLAGS := "-ldflags=-s -w"

all: $(FILES)
	go build -o main.out -tags "$(TAGS)" $(BUILD_FLAGS) $(FILES)

prql: $(FILES)
	go build -o main.out -tags "$(TAGS) prql" $(BUILD_FLAGS) $(FILES)

clean:
	rm -f main.out

.PHONY: all clean