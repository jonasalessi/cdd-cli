.PHONY: setup

## setup: configure the local clone (installs the git hooks in .githooks)
setup:
	git config core.hooksPath .githooks
	chmod +x .githooks/*
	@echo "Git hooks installed from .githooks/"
