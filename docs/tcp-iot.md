# TCP thuần cho thiết bị IoT

Project có hai listener trong cùng một tiến trình:

```text
Web / ứng dụng --HTTP :8080--┐
                             ├--> DeviceService --> repository
Thiết bị IoT ------TCP :9000-┘
```

HTTP handler và TCP server là hai cách giao tiếp khác nhau. Cả hai cùng dùng service và repository; TCP server không gọi vòng qua HTTP API.

## Giao thức mẫu

TCP là luồng byte liên tục, không tự đánh dấu ranh giới thông điệp. Template dùng **JSON mỗi dòng**, kết thúc bằng ký tự xuống dòng `\n`. Mỗi thông điệp có kích thước tối đa 64 KiB.

Mỗi thiết bị giữ một kết nối lâu dài. Đầu tiên, thiết bị xác thực bằng token:

```json
{"version":1,"type":"authenticate","device_id":"sensor-001","request_id":"auth-1","token":"TOKEN"}
```

Server phản hồi:

```json
{"version":1,"type":"ack","request_id":"auth-1","status":"authenticated"}
```

Sau đó thiết bị gửi telemetry. Template hiểu `temperature` là độ C:

```json
{"version":1,"type":"telemetry","device_id":"sensor-001","request_id":"telemetry-1","payload":{"temperature":23.5}}
```

Nếu dữ liệu hợp lệ và được lưu, server trả:

```json
{"version":1,"type":"ack","request_id":"telemetry-1","status":"stored"}
```

Thiết bị có thể gửi `ping` với `request_id`; server trả `pong`. Khi nhận lệnh, thiết bị nhận thông điệp dạng:

```json
{"version":1,"type":"command","device_id":"sensor-001","request_id":"cmd-1","payload":{"command":"restart"}}
```

Thiết bị phản hồi bằng `type: "ack"` và giữ nguyên `request_id`. Mẫu hiện ghi nhận ACK trong log; chưa lưu trạng thái thực thi lệnh.

## Chạy thử trên máy dev

Mở terminal thứ nhất, đặt cùng một token mà client thiết bị sẽ dùng, rồi chạy server:

```powershell
$env:API_TOKEN = "change-this-to-a-long-random-api-token"
$env:TCP_TOKEN = "change-this-to-a-long-random-value"
go run ./cmd/server
```

Mở terminal thứ hai, dùng đúng token đó để chạy client mẫu. Client gửi một lần đo nhiệt độ, sau đó giữ kết nối chờ lệnh:

```powershell
$env:API_TOKEN = "change-this-to-a-long-random-api-token"
$env:TCP_TOKEN = "change-this-to-a-long-random-value"
go run ./examples/tcp-device
```

Xem telemetry đã nhận:

```powershell
Invoke-RestMethod `
  -Uri http://localhost:8080/api/v1/devices/sensor-001/telemetry `
  -Headers @{ Authorization = "Bearer $env:API_TOKEN" }
```

Để gọi API thiết bị, gửi `Authorization: Bearer <API_TOKEN>`. Ví dụ gửi lệnh tới thiết bị đang online:

```powershell
Invoke-RestMethod -Method Post `
  -Uri http://localhost:8080/api/v1/devices/sensor-001/commands `
  -ContentType "application/json" `
 -Headers @{ Authorization = "Bearer $env:API_TOKEN" } `
 -Body '{"command":"restart"}'
```

API trả `202 Accepted` khi lệnh đã được ghi vào kết nối TCP, không có nghĩa thiết bị đã thực hiện xong. Thiết bị offline thì API trả `409 Conflict`.

## Cấu hình

Các biến môi trường:

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `PORT` | `8080` | Cổng HTTP |
| `API_TOKEN` | bắt buộc | Bearer token cho API telemetry/command |
| `TCP_ADDR` | `:9000` | Địa chỉ/cổng TCP server bind |
| `TCP_TOKEN` | bắt buộc | Token dùng chung để xác thực thiết bị |
| `TCP_TLS_CERT_FILE` | không bật TLS | Đường dẫn certificate server |
| `TCP_TLS_KEY_FILE` | không bật TLS | Đường dẫn private key tương ứng |

`TCP_TLS_CERT_FILE` và `TCP_TLS_KEY_FILE` phải được đặt cùng nhau. File `.env.example` chỉ là tài liệu mẫu; chương trình đọc biến môi trường của hệ điều hành chứ không tự nạp `.env`.

**Cảnh báo bảo mật:** nếu không cấu hình TLS, token TCP được truyền dưới dạng văn bản thường. API cũng dùng HTTP thuần trong template, nên `API_TOKEN` không được mã hóa trên đường truyền. Chỉ thử trên máy local hoặc mạng tin cậy; khi triển khai thật, đặt API sau HTTPS, bật TLS cho TCP, dùng token mạnh và cân nhắc danh tính/secret riêng cho từng thiết bị. Không commit token hoặc private key vào Git.

Token hiện là một giá trị dùng chung cho tất cả thiết bị, phù hợp để học giao thức chứ chưa phải quản lý danh tính thiết bị hoàn chỉnh.

## Giới hạn của bản mẫu

- Telemetry được lưu trong RAM; khởi động lại server sẽ mất dữ liệu.
- Chưa có hàng đợi lưu lệnh cho thiết bị offline.
- Chỉ hỗ trợ message `telemetry`, `ping` và `ack` từ thiết bị.
- Server tự đóng kết nối không hoạt động sau 2 phút; thiết bị thực tế nên gửi heartbeat.
- Client mẫu chỉ kết nối TCP thường, không dùng TLS.
- Cổng `:9000` bind trên mọi interface; hãy cấu hình firewall và địa chỉ phù hợp với mạng của bạn.
