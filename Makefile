PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin
XSESSIONS := /usr/share/xsessions
USER_CONFIG := $(HOME)/.config/doWM

all:
	go build -o doWM main.go
	@echo "Built successfully!"

install:
	# Install binary locally
	mkdir -p $(BINDIR)
	sudo install -m755 doWM $(BINDIR)/doWM

	# Install .desktop session file
	mkdir -p $(XSESSIONS)
	sudo install -m644 ./install/doWM.desktop $(XSESSIONS)/doWM.desktop


	@echo "Installed successfully!"

.PHONY: all install uninstall
