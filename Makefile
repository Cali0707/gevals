AGENT_BINARY_NAME = agent
MCPCHECKER_BINARY_NAME = mcpchecker
MOCK_AGENT_BINARY_NAME = functional/mock-agent

.PHONY: clean
clean:
	rm -f $(AGENT_BINARY_NAME) $(MCPCHECKER_BINARY_NAME) $(MOCK_AGENT_BINARY_NAME)
	rm -f *.zip *.bundle

.PHONY: build-agent
build-agent: clean
	go build -o $(AGENT_BINARY_NAME) ./cmd/agent

.PHONY: build-mcpchecker
build-mcpchecker: clean
	go build -o $(MCPCHECKER_BINARY_NAME) ./cmd/mcpchecker/

.PHONY: build
build: build-agent build-mcpchecker

.PHONY: test
test:
	go test -count=1 -race ./...

# Internal target - builds mock agent for functional tests
.PHONY: _build-mock-agent
_build-mock-agent:
	go build -o $(MOCK_AGENT_BINARY_NAME) ./functional/servers/agent/cmd

.PHONY: functional
functional: build _build-mock-agent ## Run functional tests
	MCPCHECKER_BINARY=$(CURDIR)/mcpchecker MOCK_AGENT_BINARY=$(CURDIR)/$(MOCK_AGENT_BINARY_NAME) go test -v -count=1 -race -tags functional ./functional/...
