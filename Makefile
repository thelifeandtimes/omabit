.PHONY: check check-all check-tend check-urbit-host build-urbit-host \
	urbit-host-version test test-js validate-plugin lint-python

check: check-tend

check-all: check-tend check-urbit-host

check-tend: validate-plugin lint-python test

check-urbit-host:
	$(MAKE) -C apps/urbit-host check

build-urbit-host:
	$(MAKE) -C apps/urbit-host build

urbit-host-version: build-urbit-host
	./apps/urbit-host/urbitctl version

validate-plugin:
	omarchy plugin validate omarchy-plugin

lint-python:
	python3 -m py_compile omarchy-plugin/transport/eyre_client.py

test: test-js
	python3 -m unittest discover -s tests -p 'test_*.py'

test-js:
	node --test tests/test_tend_model.js
