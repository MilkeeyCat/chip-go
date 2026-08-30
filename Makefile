.PHONY: wasm serve

wasm:
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" ./cmd/wasm/assets
	GOOS=js GOARCH=wasm go build -o ./cmd/wasm/assets/main.wasm ./cmd/wasm

serve:
	npx http-server ./cmd/wasm/assets
