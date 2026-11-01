# Chạy và kiểm thử

## Yêu cầu

- Cài Go phiên bản 1.22 trở lên.
- Mở terminal tại thư mục project.

## Token và xác thực

**Token** là một chuỗi bí mật dùng như "mật khẩu": server chỉ phục vụ người gọi gửi đúng token, còn thiếu hoặc sai sẽ bị từ chối.

Project có hai cổng vào, mỗi cổng dùng một token riêng:

| Token | Bảo vệ | Ai dùng | Cách gửi |
|---|---|---|---|
| `API_TOKEN` | HTTP API thiết bị (`/api/v1/devices/...`: xem telemetry, gửi lệnh) | Web, app, `examples/api-client` | Header `Authorization: Bearer <token>` |
| `TCP_TOKEN` | Kết nối TCP của thiết bị IoT (cổng 9000) | Thiết bị, `examples/tcp-device` | Trường `token` trong message `authenticate` |

Tại sao bắt buộc đặt:

- Nếu không có token, bất kỳ ai biết địa chỉ server đều đọc được dữ liệu hoặc điều khiển thiết bị.
- `internal/config/config.go` kiểm tra lúc khởi động và dừng với lỗi `API_TOKEN must be set` hoặc `TCP_TOKEN must be set` nếu thiếu.
- Không có giá trị mặc định, nên code không chứa token yếu hoặc token công khai.

Lưu ý:

- Server và client phải dùng **cùng giá trị**. Sai token thì API trả `401 unauthorized`, thiết bị TCP bị từ chối xác thực.
- Khi chạy thử local, giá trị nào cũng được, ví dụ `"abc123"`. Khi triển khai thật, dùng chuỗi dài, ngẫu nhiên và không commit lên Git.
- Biến môi trường chỉ có hiệu lực trong cửa sổ terminal bạn đặt nó, nên mỗi terminal mới phải đặt lại.
- `/health` và API user (`/api/v1/users`) không cần token trong bản mẫu hiện tại.
- Không có TLS thì token đi qua mạng dưới dạng văn bản thường; xem cảnh báo trong [hướng dẫn TCP IoT](tcp-iot.md).

### Lệnh `$env:` hoạt động như thế nào

```powershell
$env:API_TOKEN = "change-this-to-a-long-random-api-token"
$env:TCP_TOKEN = "change-this-to-a-long-random-value"
go run ./cmd/server
```

1. `$env:TÊN = "giá trị"` là cú pháp PowerShell để tạo hoặc ghi đè **biến môi trường** trong tiến trình PowerShell hiện tại.
2. Khi chạy `go run ./cmd/server`, PowerShell tạo tiến trình con. Tiến trình con thừa hưởng bản sao các biến môi trường của cha.
3. `internal/config/config.go` gọi `os.Getenv("API_TOKEN")` và `os.Getenv("TCP_TOKEN")` để đọc giá trị, sau đó `Validate()` kiểm tra chúng không rỗng.

```text
PowerShell ($env:API_TOKEN=..., $env:TCP_TOKEN=...)
   └─ go run ./cmd/server  (tiến trình con thừa hưởng biến)
        └─ os.Getenv("API_TOKEN")  →  "change-this-to-a-long-random-api-token"
```

Liên quan tới thư mục hiện tại:

- Thư mục chỉ quyết định `./cmd/server` được tính từ đâu. Biến môi trường **không gắn với thư mục**; đặt ở thư mục nào cũng được, miễn là chạy server trong **cùng cửa sổ terminal**.
- Cửa sổ terminal khác hoặc terminal mở lại sẽ không có biến này, và server báo `TCP_TOKEN must be set`.
- Biến chỉ tồn tại trong phiên terminal đó và mất khi đóng terminal; chúng không được ghi vào file.

Kiểm tra và xoá biến:

```powershell
$env:API_TOKEN                  # xem giá trị hiện tại
Remove-Item Env:API_TOKEN       # xoá biến
Remove-Item Env:TCP_TOKEN
```

### Code kiểm tra token nằm ở đâu

| Việc | Vị trí |
|---|---|
| Đọc `API_TOKEN`, `TCP_TOKEN` từ môi trường | `Load()` trong `internal/config/config.go` |
| Từ chối khởi động nếu thiếu token | `Validate()` trong `internal/config/config.go`, được gọi ở đầu `run()` trong `cmd/server/main.go` |
| Kiểm tra `API_TOKEN` của mỗi request HTTP | `authorized()` trong `internal/handler/device.go` (so sánh bằng `subtle.ConstantTimeCompare`) |
| Kiểm tra `TCP_TOKEN` của thiết bị | Xử lý message `authenticate` trong `internal/transport/tcp/server.go` |

## Chạy server

Server yêu cầu `API_TOKEN` cho các API thiết bị và `TCP_TOKEN` để xác thực thiết bị. Trong PowerShell:

```powershell
$env:API_TOKEN = "change-this-to-a-long-random-api-token"
$env:TCP_TOKEN = "change-this-to-a-long-random-value"
go run ./cmd/server
```

HTTP server mặc định lắng nghe ở cổng `8080`, TCP server ở cổng `9000`. Để đổi HTTP port, đặt biến môi trường trước khi chạy:

```powershell
$env:PORT = "9090"
go run ./cmd/server
```

Để đổi TCP address, đặt `TCP_ADDR`, ví dụ `$env:TCP_ADDR = ":9001"`. Cấu hình token và TLS được mô tả trong [hướng dẫn TCP IoT](tcp-iot.md).

Nhấn Ctrl+C để dừng server. Chương trình sẽ chờ một khoảng thời gian ngắn để các request đang xử lý kết thúc.

## Gọi thử API bằng PowerShell

Mở terminal thứ hai trong khi server vẫn đang chạy:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Tạo user:

```powershell
Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/api/v1/users `
  -ContentType "application/json" `
  -Body '{"name":"An"}'
```

Lấy danh sách:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/users
```

User chỉ được lưu trong bộ nhớ. Nếu khởi động lại server, danh sách sẽ trở về rỗng.

Các API thiết bị yêu cầu header `Authorization: Bearer <API_TOKEN>`. Ví dụ:

```powershell
$env:API_TOKEN = "change-this-to-a-long-random-api-token"
Invoke-RestMethod `
  -Uri http://localhost:8080/api/v1/devices/sensor-001/telemetry `
  -Headers @{ Authorization = "Bearer $env:API_TOKEN" }
```

## Dùng client API

`examples/api-client` là CLI gọi HTTP API. Khi server đang chạy, mở terminal thứ hai:

```powershell
$env:API_TOKEN = "change-this-to-a-long-random-api-token"   # cùng giá trị với server
go run ./examples/api-client health
go run ./examples/api-client create-user An
go run ./examples/api-client users
go run ./examples/api-client telemetry sensor-001
go run ./examples/api-client command sensor-001 restart
```

Đặt `API_URL` nếu server không chạy ở `http://localhost:8080`. Client thiết bị TCP nằm ở `examples/tcp-device`.

## Chạy test

```powershell
go test ./...
```

Lệnh này chạy các test trong module. Test mẫu tại `internal/service/user_test.go` kiểm tra service từ chối tên trống và tạo user hợp lệ.

Các lệnh Makefile như `make run` và `make test` ép Go dùng `-mod=vendor`. Vì template hiện chưa có dependency ngoài và chưa có thư mục `vendor/`, hãy dùng trực tiếp các lệnh `go run` và `go test` phía trên. Sau khi thêm dependency, chuẩn bị `vendor/` theo [hướng dẫn offline](offline-vendor.md), rồi có thể dùng các target Makefile.

Trên Windows, `make` chỉ dùng được nếu bạn đã cài công cụ cung cấp lệnh Make. Nếu chưa có, chạy trực tiếp các lệnh `go` ở trên.

## Build, task và debug trong VS Code

Project có sẵn cấu hình trong thư mục `.vscode`:

- Nhấn **Ctrl+Shift+B** để chạy task build mặc định, tạo `server.exe` ở thư mục gốc.
- Mở **Terminal > Run Task...** để chạy `Go: Test all packages`, `Go: Vet all packages` hoặc `Go: Run server`.
- Mở `cmd/server/main.go`, đặt breakpoint và nhấn **F5** để debug cấu hình `Debug Server`. Nhấn **Shift+F5** để dừng.

Cài extension **Go** của Go Team (`golang.go`) và chấp nhận cài Delve nếu VS Code yêu cầu. Task chạy server và cấu hình debug đặt token mẫu dành cho local development; hãy thay chúng bằng giá trị riêng nếu cần gọi API thiết bị. Không dùng token mẫu trong môi trường thật.

## Một số lệnh hữu ích

```powershell
go fmt ./...       # định dạng code Go
go vet ./...       # kiểm tra một số lỗi phổ biến
go test ./...      # chạy test
```
