# ServerGO

## Cấu trúc

```
ServerGO/
├── cmd/server/main.go     # Điểm vào: khởi tạo, nối dependency, graceful shutdown
├── internal/              # Code riêng của dự án (không import được từ bên ngoài)
│   ├── config/            # Đọc cấu hình từ biến môi trường
│   ├── handler/           # Tầng HTTP: parse request, trả response
│   ├── service/           # Tầng nghiệp vụ (business logic)
│   ├── repository/        # Tầng truy cập dữ liệu (DB, cache...)
│   ├── model/             # Struct dữ liệu dùng chung
│   └── server/            # Router + middleware
├── pkg/                   # (tùy chọn) code dùng lại được cho dự án khác
├── configs/               # (tùy chọn) file cấu hình
├── migrations/            # (tùy chọn) SQL migration
├── Makefile
└── .env.example
```

Luồng: `handler -> service -> repository`. Mỗi tầng chỉ gọi tầng bên dưới.

## Chạy

```
go run ./cmd/server      # hoặc: make run
go test ./...            # hoặc: make test
```

Thử:
```
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/users -d "{\"name\":\"An\"}"
curl http://localhost:8080/api/v1/users
```

## Môi trường cách ly mạng (vendor)

Thư mục `vendor/` chứa toàn bộ source của dependency và được commit cùng repo, nên máy dev không cần mạng. Makefile đã dùng `-mod=vendor`.

**Trên máy có mạng (trung gian):**
```
go get github.com/ten/goi@v1.2.3   # thêm dependency
make vendor                        # = go mod tidy + go mod vendor
```
Sau đó chuyển cả repo (gồm `go.mod`, `go.sum`, `vendor/`) sang máy dev (git/USB/kho nội bộ).

**Trên máy dev (không mạng):**
```
make build    # go build -mod=vendor
make test
```

Lưu ý:
- Không chạy `go get` / `go mod tidy` trên máy cách ly; chỉ dùng `-mod=vendor`.
- Luôn commit `vendor/` và `go.sum`; sau mỗi lần đổi dependency phải chạy lại `make vendor`.
- Khi `go.mod` có `go 1.14+` và tồn tại `vendor/`, Go tự dùng vendor mặc định, nhưng vẫn nên ghi `-mod=vendor` để lỗi hiện rõ nếu vendor lệch.
- Tùy chọn: `go env -w GOFLAGS=-mod=vendor GOPROXY=off` để chặn hẳn việc tải từ mạng.
- `make verify-vendor` kiểm tra vendor khớp với `go.mod` (chạy khi có mạng/module cache).
- Dự án mẫu hiện chỉ dùng thư viện chuẩn nên chưa có `vendor/`; nó sẽ xuất hiện khi bạn thêm dependency đầu tiên.

Cần Go >= 1.22. Khi đổi tên dự án, sửa `module` trong `go.mod` và các import `servergo/...`.
