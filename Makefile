all: mocks

mocks:
	mockery -dir ./fs -name FS
	mockery -dir ./walker -name Walker


COVERAGE_TMP_FILE=shrunkcoverate.out
COVERAGE_OUTPUT_FILE=coverage.html

coverage:
	- rm $(COVERAGE_OUTPUT_FILE)
	echo "Run tests..."
	go test -v ./... -coverprofile $(COVERAGE_TMP_FILE) cover
	echo "Generate code coverage..."
	go tool cover -html=$(COVERAGE_TMP_FILE) -o $(COVERAGE_OUTPUT_FILE)
	rm $(COVERAGE_TMP_FILE)
