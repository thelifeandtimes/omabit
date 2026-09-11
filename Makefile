.PHONY: check check-all check-tend check-tend-release check-tend-migrations check-urbit-host \
	build-urbit-host dist-tend urbit-host-version test test-js \
	validate-plugin lint-python

check: check-tend

check-all: check-tend check-urbit-host

check-tend: validate-plugin lint-python test

check-urbit-host:
	$(MAKE) -C apps/urbit-host check

check-tend-release:
	scripts/check-tend-release.sh

check-tend-migrations:
	@test -n "$(TEND_TEST_PIER)" || { echo "set TEND_TEST_PIER to a running fake-ship pier" >&2; exit 64; }
	scripts/check-tend-migrations.sh "$(TEND_TEST_PIER)"

dist-tend:
	scripts/build-tend-release.sh

build-urbit-host:
	$(MAKE) -C apps/urbit-host build

urbit-host-version: build-urbit-host
	./apps/urbit-host/urbitctl version

validate-plugin:
	omarchy plugin validate omarchy-plugin

lint-python:
	python3 -m py_compile omarchy-plugin/transport/eyre_client.py
	python3 -m py_compile bin/omabit

test: test-js
	python3 -m unittest discover -s tests -p 'test_*.py'

test-js:
	node --test tests/test_tend_*.js
