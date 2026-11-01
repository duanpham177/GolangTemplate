# Luồng xử lý một request

Tài liệu này lần theo ví dụ tạo user để cho thấy các phần trong project phối hợp như thế nào.

## Request mẫu

Client gửi đến server:

```http
POST /api/v1/users
Content-Type: application/json

{"name":"An"}
```

## Từ router đến dữ liệu

### 1. Router chọn handler

[internal/server/router.go](../internal/server/router.go) đăng ký route `POST /api/v1/users` và chuyển request đến `users.Create`.

Router chỉ định tuyến: dựa trên phương thức HTTP và đường dẫn để chọn hàm xử lý. Nó không nên chứa toàn bộ quy tắc của ứng dụng.

### 2. Handler đọc HTTP

[internal/handler/user.go](../internal/handler/user.go) giải mã phần JSON trong body thành một struct có trường `Name`. Handler gọi service với tên nhận được.

Nếu body không phải JSON hợp lệ, handler trả `400 Bad Request`. Nếu request hợp lệ, handler tiếp tục gọi tầng nghiệp vụ.

### 3. Service kiểm tra quy tắc

[internal/service/user.go](../internal/service/user.go) cắt khoảng trắng đầu/cuối tên. Nếu tên rỗng, service trả về `ErrInvalidName`; handler chuyển lỗi đó thành response `400`.

Quy tắc như “tên không được rỗng” thuộc về service, không phải router. Nhờ vậy, dù service được gọi từ HTTP hay một nơi khác, quy tắc vẫn giống nhau.

### 4. Repository lưu dữ liệu

Nếu tên hợp lệ, service gọi repository để tạo user. Repository hiện dùng một slice trong bộ nhớ và gán ID tăng dần.

Điều này chỉ phù hợp làm ví dụ: dữ liệu mất khi chương trình dừng. Muốn lưu bền vững cần thay implementation bằng database hoặc kho lưu trữ khác.

### 5. Handler trả response

Handler chuyển user thành JSON và trả mã `201 Created`, ví dụ:

```json
{"id":1,"name":"An"}
```

## Luồng lấy danh sách

Với `GET /api/v1/users`, router gọi `users.List`; handler gọi service; service gọi repository; cuối cùng handler trả danh sách JSON với mã `200 OK`.

## Middleware log

`NewRouter` bọc router trong middleware `logging`. Middleware đo thời gian trước và sau khi router xử lý request, rồi ghi log. Middleware có thể dùng cho những việc áp dụng chung như log, đo thời gian hoặc xác thực.

## Luồng telemetry qua TCP

Thiết bị IoT không gọi handler HTTP. Nó mở kết nối TCP, xác thực, rồi gửi JSON mỗi dòng. TCP server trong `internal/transport/tcp/` kiểm tra thông điệp và gọi cùng `DeviceService` để lưu telemetry.

```text
Thiết bị -> TCP transport -> DeviceService -> TelemetryRepository
Web API  -> HTTP handler  -> DeviceService -> TelemetryRepository
```

Web có thể đọc dữ liệu qua `GET /api/v1/devices/{deviceID}/telemetry`. Lệnh điều khiển đi ngược lại: HTTP handler gọi service, service yêu cầu TCP connection manager gửi lệnh đến thiết bị đang online. Xem [hướng dẫn TCP IoT](tcp-iot.md) để biết định dạng message và cách chạy client mẫu.

## Tóm tắt

```text
POST /api/v1/users
  -> router chọn UserHandler.Create
  -> handler giải mã JSON
  -> service kiểm tra tên
  -> repository tạo user
  -> handler trả JSON + HTTP 201
```
