PROTO_DIR=proto
GEN_DIR=gen

.PHONY: proto
proto:
	buf generate

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: run-usersvc
run-usersvc:
	go run ./services/usersvc

.PHONY: run-ordersvc
run-ordersvc:
	go run ./services/ordersvc

.PHONY: run-inventorysvc
run-inventorysvc:
	go run ./services/inventorysvc
