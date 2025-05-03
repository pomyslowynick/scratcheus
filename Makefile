run:
	go run main.go

tidy:
	go mod tidy

generate_lexer:
	golex -o=parser/openmetrics-lexer.l.go parser/openmetrics-lexer.l
