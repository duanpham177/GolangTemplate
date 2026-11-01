# Luồng code, các đối tượng và tư duy khi gặp vấn đề mới

Các tài liệu khác cho biết **từng file làm gì**. Tài liệu này trả lời ba câu hỏi còn thiếu:

1. Khi chương trình chạy, **các đối tượng nào được tạo và nối với nhau** như thế nào?
2. **Dữ liệu đi qua code** theo luồng nào trong từng tình huống?
3. Khi gặp một yêu cầu mới, **nên suy nghĩ và bắt đầu từ đâu**?

## 1. Bản đồ các đối tượng

Mỗi ô là một kiểu (struct hoặc interface) trong code. Mũi tên `──►` nghĩa là "giữ một tham chiếu tới và gọi".

```text
                    ┌───────────────────────────── cmd/server/main.go ─────────────────────────────┐
                    │  tạo tất cả đối tượng bên dưới và nối chúng lại (dependency injection)       │
                    └───────────────────────────────────────────────────────────────────────────────┘

  HTTP (cổng 8080)                                                    TCP (cổng 9000)
  ────────────────                                                    ───────────────
  http.Server                                                         tcp.Server
     │                                                                   │  (token, TLS, mỗi kết nối 1 goroutine)
     ▼                                                                   │
  Router (mux + middleware logging)                                      │
     │                                                                   │
     ├─► UserHandler ───────► UserService ───► UserRepository            │
     │                                          (interface)              │
     │                                              ▲                    │
     │                                       userMemory (RAM)            │
     │                                                                   │
     └─► DeviceHandler ─────► DeviceService ◄── TelemetryRecorder ───────┘
          (giữ API_TOKEN)        │      │         (interface, DeviceService thoả mãn)
                                 │      │
                                 │      └──► CommandSender (interface) ◄── tcp.Manager
                                 │                                           │ map deviceID -> clientConn
                                 ▼                                           │ (RWMutex)
                        TelemetryRepository (interface)                      ▼
                                 ▲                                      clientConn
                          telemetryMemory (RAM)                         (net.Conn + writeMu)
```

Ba điều cần nhớ từ sơ đồ:

- **`DeviceService` là điểm gặp nhau.** HTTP handler và TCP server cùng gọi nó, nên quy tắc nghiệp vụ chỉ viết một lần.
- **Phụ thuộc đi theo interface.** `DeviceService` chỉ biết `TelemetryRepository` và `CommandSender`, không biết RAM hay TCP. Nhờ vậy thay RAM bằng database, hoặc thay TCP bằng MQTT, mà không phải sửa service.
- **`main.go` là nơi duy nhất biết toàn bộ.** Các package khác chỉ biết phần mình cần.

### Các kiểu dữ liệu chính

| Kiểu | File | Vai trò |
|---|---|---|
| `model.User` | `internal/model/user.go` | Dữ liệu một user (`ID`, `Name`) |
| `model.Telemetry` | `internal/model/telemetry.go` | Một lần đo: `DeviceID`, `Temperature`, `RecordedAt` |
| `config.Config` | `internal/config/config.go` | Cấu hình đọc từ biến môi trường |
| `tcp.Message` | `internal/transport/tcp/protocol.go` | Một dòng JSON thiết bị gửi lên (`type`, `device_id`, `request_id`, `token`, `payload`) |
| `tcp.response` | `internal/transport/tcp/protocol.go` | Một dòng JSON server trả về (`type`, `status`, `error`) |
| `tcp.clientConn` | `internal/transport/tcp/manager.go` | Một kết nối thiết bị, kèm mutex để ghi tuần tự |

## 2. Luồng khởi động

`run()` trong `cmd/server/main.go` làm theo thứ tự sau:

```text
1. Load()           đọc biến môi trường vào Config
2. Validate()       thiếu API_TOKEN / TCP_TOKEN -> trả lỗi, chương trình dừng (exit 1)
3. Tạo từ dưới lên: repository -> service -> handler
                    tcp.Manager -> DeviceService (Manager là CommandSender)
                    tcp.Server nhận DeviceService (là TelemetryRecorder) và Manager
4. Chạy 2 goroutine: HTTP ListenAndServe, TCP Serve(ctx)
5. Chờ trong select: Ctrl+C / lỗi HTTP / TCP dừng
6. Tắt có trật tự:   cancel(ctx) -> srv.Shutdown (tối đa 10 giây) -> chờ TCP dừng
```

Quy tắc xây dựng: **tạo từ phụ thuộc thấp nhất lên cao nhất**, vì đối tượng trên cần nhận đối tượng dưới làm tham số. Đó là lý do `NewDeviceService(telemetryRepo, tcpManager)` phải đứng sau hai đối tượng kia.

## 3. Các luồng dữ liệu

### 3.1 Tạo user qua HTTP

```text
POST /api/v1/users {"name":"An"}
  Router            chọn UserHandler.Create theo "POST /api/v1/users"
  UserHandler       giải mã JSON ─ lỗi ─► 400 "invalid json"
  UserService       TrimSpace, rỗng ─ lỗi ─► ErrInvalidName ─► handler đổi thành 400
  UserRepository    gán ID, lưu vào slice (có khoá mutex)
  UserHandler       201 {"id":1,"name":"An"}
```

### 3.2 Xem telemetry qua HTTP (có token)

```text
GET /api/v1/devices/sensor-001/telemetry   Authorization: Bearer <API_TOKEN>
  DeviceHandler.authorized   sai/thiếu token ─► 401   (dừng tại đây)
  DeviceService.ListTelemetry  TrimSpace deviceID, rỗng ─► ErrInvalidDeviceID ─► 400
  TelemetryRepository.ListByDevice  lọc các bản ghi theo device_id
  DeviceHandler              200 + JSON
```

### 3.3 Thiết bị gửi telemetry qua TCP

```text
Kết nối TCP mới ─► goroutine riêng cho kết nối này
  Chờ dòng đầu tối đa 10 giây
  authenticate? token đúng? ─ không ─► "authentication failed", đóng kết nối
  Manager.register(deviceID, clientConn)    <- từ đây thiết bị được coi là "online"
  Vòng lặp (mỗi vòng đặt hạn đọc 2 phút):
      đọc 1 dòng JSON ─► decodeMessage ─► kiểm tra device_id khớp, request_id hợp lệ
      type = telemetry ─► handleTelemetry ─► DeviceService.RecordTelemetry ─► repository.Save ─► ack "stored"
      type = ping      ─► pong
      type = ack       ─► ghi log (xác nhận lệnh)
      khác             ─► error "unsupported message type"
  Thoát vòng lặp ─► Manager.unregister, đóng kết nối
```

### 3.4 Gửi lệnh tới thiết bị (HTTP rồi TCP)

```text
POST /api/v1/devices/sensor-001/commands {"command":"restart"}   Bearer <API_TOKEN>
  DeviceHandler        kiểm tra token; giải mã JSON (tối đa 4 KiB, cấm trường lạ)
  DeviceService        kiểm tra deviceID và command không rỗng
  CommandSender        (thực tế là tcp.Manager) tìm clientConn theo deviceID
      không có ─► ErrDeviceOffline ─► handler đổi thành 409
      có       ─► clientConn.write (khoá writeMu) ─► thiết bị nhận message "command"
  DeviceHandler        202 Accepted   (nghĩa là đã ghi vào kết nối, chưa chắc thiết bị đã làm xong)
```

### 3.5 Lỗi đi ngược lên như thế nào

Mỗi tầng tự quyết định lỗi nào mình trả, tầng trên **dịch** lỗi sang ngôn ngữ của nó:

```text
repository / manager   trả lỗi hoặc sentinel (ví dụ service.ErrDeviceOffline)
service                kiểm tra quy tắc, trả sentinel: ErrInvalidDeviceID, ErrInvalidCommand, ErrDeviceOffline...
handler                errors.Is(err, ...) ─► chọn HTTP status (400, 409, 500)
tcp server             errors.Is(err, ...) ─► chọn message "error" trả cho thiết bị
```

Service không biết HTTP status là gì. Cùng một lỗi service có thể thành `400` ở HTTP và thành `{"type":"error"}` ở TCP.

## 4. Đồng thời (concurrency): vì sao có mutex, goroutine, channel

Nhiều request và nhiều thiết bị chạy cùng lúc, nên mọi dữ liệu dùng chung phải được bảo vệ:

| Dữ liệu dùng chung | Bảo vệ bằng | Vì sao |
|---|---|---|
| Slice user / telemetry trong RAM | `sync.RWMutex` trong repository | Nhiều goroutine đọc và ghi cùng lúc |
| Map thiết bị đang online | `sync.RWMutex` trong `tcp.Manager` | Thiết bị vào/ra liên tục, trong khi HTTP đang tra cứu |
| Ghi vào một kết nối TCP | `writeMu` trong `clientConn` | Server (trả ack) và HTTP (gửi lệnh) có thể ghi cùng lúc, không được xen kẽ byte |
| Dừng server | `context` + channel `stop`, `httpErr`, `tcpDone` | Báo cho các goroutine biết lúc nào dừng |

Mỗi kết nối TCP có **một goroutine riêng** nên một thiết bị chậm không chặn thiết bị khác. HTTP cũng vậy: Go tự tạo goroutine cho mỗi request.

## 5. Tư duy khi gặp vấn đề mới

### 5.1 Năm câu hỏi đầu tiên

Trước khi viết code, trả lời:

1. **Ai kích hoạt?** (người dùng qua HTTP, thiết bị qua TCP, hay chính server theo lịch)
2. **Dữ liệu vào và ra là gì?** (JSON, trường nào, kiểu gì, bắt buộc hay tuỳ chọn)
3. **Quy tắc nghiệp vụ là gì?** (hợp lệ khi nào, lỗi nào có thể xảy ra)
4. **Dữ liệu sống ở đâu?** (RAM, database, hay chỉ đi qua)
5. **Ai được phép làm việc này?** (cần token nào, có cần phân quyền thêm không)

### 5.2 Mỗi loại việc thuộc về tầng nào

| Việc cần làm | Đặt ở | Ví dụ |
|---|---|---|
| Đọc request, chọn HTTP status | `handler` | Đọc `{id}`, trả `404` |
| Quy tắc, kiểm tra, quyết định | `service` | "nhiệt độ phải là số hữu hạn" |
| Lưu và truy vấn dữ liệu | `repository` | `Save`, `ListByDevice` |
| Hình dạng dữ liệu | `model` | Thêm trường vào `Telemetry` |
| Định dạng gói tin, kết nối, timeout | `transport/tcp` | Thêm loại message mới |
| Nối các phần, đọc cấu hình | `main.go`, `config` | Thêm biến môi trường |

Nếu bạn không biết đặt đâu, hãy hỏi: **"Nếu đổi HTTP thành giao thức khác, phần này có phải đổi không?"** Có thì nó thuộc handler hoặc transport. Không thì nó thuộc service hoặc repository.

### 5.3 Quy trình làm một tính năng

1. **Viết hợp đồng trước:** chọn tên hàm, tham số, giá trị trả về và các lỗi (sentinel error).
2. **Đi từ trong ra ngoài:** `model` → `repository` → `service` → `handler`/`transport` → `router` → `main.go` nếu cần nối thêm.
3. **Viết test cho service trước** vì nó thuần logic, chạy nhanh và không cần mạng.
4. **Chạy `go build ./...` và `go vet ./...` sau mỗi bước nhỏ.** Trình biên dịch sẽ chỉ ra nơi nào bị ảnh hưởng bởi thay đổi.
5. **Thử thật** bằng server và client (xem [Chạy và kiểm thử](running-testing.md)).

### 5.4 Ví dụ 1: thêm độ ẩm vào telemetry

Yêu cầu: thiết bị gửi thêm `humidity` (độ ẩm, %).

| Bước | Quyết định và việc làm |
|---|---|
| Hỏi | Thiết bị cũ chưa gửi `humidity`, có bị hỏng không? → Dùng con trỏ `*float64` để biết "có gửi hay không". |
| `model` | Thêm `Humidity *float64` vào `Telemetry` (`json:"humidity,omitempty"`). |
| `service` | Sửa `RecordTelemetry` nhận thêm độ ẩm. Thêm quy tắc: nếu có thì phải trong khoảng 0 đến 100, ngược lại trả lỗi mới `ErrInvalidTelemetry`. |
| `transport/tcp` | Trong `handleTelemetry`, đọc thêm `humidity` từ payload và truyền xuống service. |
| `repository` | **Không phải sửa**, vì nó chỉ lưu nguyên `model.Telemetry`. |
| `handler` | **Không phải sửa**, vì nó trả nguyên `model.Telemetry` dưới dạng JSON. |
| Hệ quả | Đổi chữ ký `RecordTelemetry` sẽ làm interface `TelemetryRecorder` trong `server.go` và các test gọi hàm này không biên dịch. Trình biên dịch báo ngay, sửa theo từng chỗ. |
| Test | Thêm test service cho độ ẩm hợp lệ, vượt khoảng, và trường hợp không gửi. |

Điểm rút ra: một thay đổi dữ liệu chạm vài tầng, nhưng các tầng chỉ "chuyển tiếp" không phải sửa.

### 5.5 Ví dụ 2: thay RAM bằng database

Yêu cầu: telemetry phải còn sau khi khởi động lại server.

1. Giữ nguyên interface `TelemetryRepository` (`Save`, `ListByDevice`).
2. Viết một implementation mới (ví dụ `telemetryPostgres`) trong `internal/repository/`.
3. Thêm cấu hình kết nối vào `config`, và thay `repository.NewTelemetryMemory()` trong `main.go` bằng hàm tạo mới.
4. `service`, `handler`, `transport` **không đổi**.
5. Lưu ý: `ListByDevice` hiện không trả lỗi. Database có thể lỗi, nên cân nhắc đổi chữ ký để trả `error`. Đây là lúc thay đổi lan lên service và handler.

Điểm rút ra: interface giúp thay công nghệ lưu trữ mà ít sửa nhất, nhưng hợp đồng phải đủ để biểu diễn lỗi của công nghệ mới.

### 5.6 Ví dụ 3: thêm loại message TCP mới

Yêu cầu: thiết bị gửi `status` (pin, tín hiệu).

1. Thêm `case "status":` vào `switch message.Type` trong `handleConnection`.
2. Viết hàm xử lý giống `handleTelemetry`: kiểm tra payload, gọi service, trả `ack` hoặc `error`.
3. Quy tắc và lưu trữ thuộc service và repository, không viết trong transport.
4. Cập nhật mô tả giao thức trong [tcp-iot.md](tcp-iot.md) và thêm test trong `server_test.go`.

## 6. Cách gỡ lỗi theo luồng

Khi có lỗi, **xác định luồng bị đứt ở tầng nào** thay vì đoán:

| Triệu chứng | Nghĩ tới | Kiểm tra |
|---|---|---|
| Server không khởi động, báo `... must be set` | `Validate()` | Đã đặt biến môi trường trong đúng terminal chưa |
| API trả `401` | `DeviceHandler.authorized` | Header có tiền tố `Bearer ` không, token khớp server không |
| API trả `409 device is not connected` | `Manager` không có thiết bị | Thiết bị đã xác thực TCP chưa, `device_id` có khớp không |
| API trả `400` | handler hoặc service từ chối | Đọc thông báo `error` trong body, tìm sentinel tương ứng |
| Thiết bị bị ngắt kết nối | timeout hoặc xác thực | 10 giây chưa gửi `authenticate`, 2 phút không gửi gì (cần `ping`), sai token |
| Dữ liệu mất sau khi chạy lại | repository trong RAM | Bình thường với bản mẫu |

Cách làm: bắt đầu từ **điểm vào** của luồng (router hoặc `handleConnection`), đặt breakpoint (F5 trong VS Code, xem [Chạy và kiểm thử](running-testing.md)) hoặc thêm log, rồi đi theo từng tầng đến khi giá trị sai xuất hiện. Log của server có sẵn ở dạng JSON, ví dụ `TCP device connected` kèm `device_id`.

## 7. Checklist trước khi coi là xong

- [ ] Quy tắc nghiệp vụ nằm trong service, không nằm trong handler hay transport.
- [ ] Service không import `net/http`, không biết status code.
- [ ] Mọi lỗi được xử lý hoặc trả lên, không bỏ qua.
- [ ] Dữ liệu dùng chung có mutex, hoặc không bị chia sẻ.
- [ ] Đầu vào không tin cậy được kiểm tra: kích thước, kiểu, rỗng.
- [ ] Đường có token đã được thử với trường hợp thiếu, sai và đúng token.
- [ ] Có test cho hành vi quan trọng; `go build ./...`, `go vet ./...` và `go test ./...` đều qua.
- [ ] Cập nhật tài liệu nếu giao thức hoặc API thay đổi.

## Đọc tiếp

- [Tổng quan cấu trúc project](project-structure.md)
- [Luồng xử lý một request](request-lifecycle.md)
- [Thêm API mới](adding-feature.md)
- [TCP thuần cho IoT](tcp-iot.md)
