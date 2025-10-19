.DEFAULT_GOAL := dev

tw:
	@npx @tailwindcss/cli -i input.css -o ./public/static/css/tw.css --watch

run:
	go run cmd/simpleci/main.go

dev:
	@templ generate -watch -proxyport=7332 -proxy="http://localhost:8080" -open-browser=false -cmd="go run cmd/simpleci/main.go"
