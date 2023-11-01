
:test
test: fmt ## Run Tests
	go test `go list ./... | grep -v test/e2e | grep -v test/coexistence` -timeout 60m -race -coverprofile coverage.txt ${GOTEST_VERBOSE}
goto :EOF