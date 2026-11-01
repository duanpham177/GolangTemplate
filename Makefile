.PHONY: run build test fmt vet tidy vendor verify-vendor

# Offline: luôn dùng thư mục vendor/, không truy cập mạng
GOFLAGS_OFFLINE = -mod=vendor

run:
	go run $(GOFLAGS_OFFLINE) ./cmd/server

build:
	go build $(GOFLAGS_OFFLINE) -o bin/server ./cmd/server

test:
	go test $(GOFLAGS_OFFLINE) ./...

fmt:
	go fmt ./...

vet:
	go vet $(GOFLAGS_OFFLINE) ./...

# Chạy trên máy CÓ mạng (máy trung gian): cập nhật go.mod/go.sum
tidy:
	go mod tidy

# Chạy trên máy CÓ mạng: tải dependency vào vendor/
vendor:
	go mod tidy
	go mod vendor

# Kiểm tra vendor/ khớp với go.mod
verify-vendor:
	go mod verify
	go build -mod=vendor ./...
