# Làm việc offline với vendor

Hướng dẫn này dành cho trường hợp máy phát triển không được kết nối mạng. Dependency cần được chuẩn bị trước trên một máy được phép truy cập mạng.

## `vendor/` là gì?

`vendor/` chứa bản sao source code của các module bên ngoài mà project sử dụng. Khi build ở chế độ vendor, Go lấy dependency từ thư mục này thay vì tải từ mạng.

Template hiện chưa có `vendor/` vì nó chỉ dùng thư viện chuẩn của Go. Thư viện chuẩn được cài cùng Go, không cần vendor.

## Chuẩn bị trên máy có mạng

Khi thêm một dependency, dùng máy trung gian được phép truy cập nguồn dependency:

```powershell
go get github.com/ten/goi@v1.2.3
go mod tidy
go mod vendor
```

Hoặc dùng target Makefile:

```powershell
make vendor
```

Target này chạy `go mod tidy` và `go mod vendor`, vì vậy **không chạy nó trên máy cách ly**.

Sau khi chuẩn bị, đưa các thay đổi sau vào gói chuyển nội bộ, thường qua Git hoặc phương tiện được tổ chức cho phép:

- `go.mod`
- `go.sum`
- thư mục `vendor/`
- code sử dụng dependency mới

## Build và test trên máy cách ly

Makefile cấu hình build, test và run dùng `-mod=vendor`:

```powershell
make build
make test
```

Hoặc chạy trực tiếp:

```powershell
go build -mod=vendor ./...
go test -mod=vendor ./...
```

Không chạy `go get` hoặc `go mod tidy` trên máy cách ly. Khi cần thêm hoặc nâng cấp dependency, quay lại quy trình chuẩn bị trên máy có mạng, rồi chuyển lại `go.mod`, `go.sum` và `vendor/`.

## Kiểm tra tính nhất quán

Sau khi cập nhật dependency trên máy chuẩn bị, có thể chạy:

```powershell
go mod verify
go build -mod=vendor ./...
```

`go mod verify` kiểm tra module trong module cache; lệnh build với `-mod=vendor` kiểm tra project có thể build ở chế độ vendor. Target `make verify-vendor` chạy hai lệnh này.

Chỉ commit hoặc chuyển dependency từ nguồn đã được tổ chức phê duyệt. Vendor giúp build offline; nó không tự xác nhận dependency có an toàn hay được phép sử dụng.
