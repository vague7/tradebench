BIN_DIR := ./bin
BIN := $(BIN_DIR)/ref-engine

all: $(BIN)

$(BIN): engine.cpp engine.h
	mkdir -p $(BIN_DIR)
	g++ -std=c++17 -O2 -Wall -Wextra -o $(BIN) engine.cpp

clean:
	rm -rf $(BIN_DIR)

.PHONY: all clean
