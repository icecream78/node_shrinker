WIP project for removing unnecessery files from node_modules directory

for calculating code coverage
go test -v ./... -coverprofile shrunkcoverate.out cover && go tool cover -html=shrunkcoverate.out -o cover.html && firefox cover.html

for generating mocks
mockery -dir ./fs -name FS
mockery -dir ./walker -name Walker
