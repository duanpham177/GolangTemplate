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
│   ├── server/            # HTTP router + middleware
│   └── transport/tcp/     # TCP thuần cho thiết bị IoT
├── pkg/                   # (tùy chọn) code dùng lại được cho dự án khác
├── configs/               # (tùy chọn) file cấu hình
├── migrations/            # (tùy chọn) SQL migration
├── docs/                  # Tài liệu hướng dẫn project
├── Makefile
└── .env.example
```

Luồng nghiệp vụ: `HTTP handler / TCP transport -> service -> repository`.

## Tài liệu học project

Nếu bạn mới học Go, hãy đọc theo thứ tự:

1. [Tổng quan cấu trúc project](docs/project-structure.md)
2. [Luồng xử lý một request](docs/request-lifecycle.md)
3. [Hướng dẫn thêm API mới](docs/adding-feature.md)
4. [Chạy và kiểm thử](docs/running-testing.md)
5. [Làm việc offline với vendor](docs/offline-vendor.md)
6. [TCP thuần cho IoT](docs/tcp-iot.md)
7. [Bảng thuật ngữ cơ bản](docs/glossary.md)
8. [Luồng code, đối tượng và tư duy](docs/code-flow-and-thinking.md)

Mục lục đầy đủ nằm ở [docs/README.md](docs/README.md).

## Chạy

Trong PowerShell, đặt API token và token thiết bị trước khi chạy:

```powershell
$env:API_TOKEN = "change-this-to-a-long-random-api-token"
$env:TCP_TOKEN = "change-this-to-a-long-random-value"
go run ./cmd/server
```

Server HTTP chạy ở cổng `8080`; TCP server mặc định nghe ở cổng `9000`.
API thiết bị cần header `Authorization: Bearer <API_TOKEN>`. Chi tiết giao thức và ví dụ thiết bị nằm trong [docs/tcp-iot.md](docs/tcp-iot.md).

Chạy test trong một terminal riêng:

```powershell
go test ./...
```

Thử:
```
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/users -d "{\"name\":\"An\"}"
curl http://localhost:8080/api/v1/users
```

## Môi trường cách ly mạng (vendor)

Hướng dẫn chuẩn bị dependency trên máy có mạng và build trên máy cách ly nằm ở [docs/offline-vendor.md](docs/offline-vendor.md). Project mẫu hiện chỉ dùng thư viện chuẩn nên chưa có thư mục `vendor/`.

Cần Go >= 1.22. Khi đổi tên dự án, sửa `module` trong `go.mod` và các import `servergo/...`.
