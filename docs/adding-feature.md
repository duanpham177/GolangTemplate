# Hướng dẫn thêm API mới

Ví dụ trong tài liệu này là thêm API `GET /api/v1/users/{id}` để xem một user. Mục đích là minh họa cách các tầng phối hợp; template hiện chưa có API này.

## Trước khi bắt đầu

Hãy xác định:

1. Client gửi phương thức và đường dẫn nào?
2. Request cần dữ liệu gì?
3. Quy tắc nghiệp vụ là gì?
4. Dữ liệu được đọc hoặc lưu ở đâu?
5. Khi thành công hoặc lỗi, API trả gì?

Không phải API nào cũng cần thay đổi tất cả các tầng. Ví dụ endpoint health đơn giản có thể chỉ cần handler và router.

Nếu tính năng đến từ thiết bị TCP, hãy xử lý giao thức trong `internal/transport/tcp/`, sau đó gọi service dùng chung. Không gọi ngược qua HTTP handler.

## Các bước thường gặp

### 1. Xác định dữ liệu

Nếu endpoint cần dữ liệu mới hoặc cần trả thêm thông tin, kiểm tra xem kiểu trong `internal/model/` đã phù hợp chưa. Ví dụ hiện tại là [User](../internal/model/user.go).

### 2. Thêm thao tác lưu trữ khi cần

Nếu tính năng cần đọc hoặc ghi dữ liệu, thêm thao tác tương ứng vào `UserRepository` và implementation trong `internal/repository/`.

Repository hiện lưu trong bộ nhớ, nên cần cập nhật cả hai phần. Nếu sau này có database, implementation mới sẽ thực hiện truy vấn database.

### 3. Thêm quy tắc vào service

Thêm phương thức vào `UserService` trong `internal/service/`. Đặt kiểm tra dữ liệu và quy tắc nghiệp vụ ở đây, rồi trả lỗi rõ ràng nếu đầu vào không hợp lệ.

Đừng để handler gọi repository trực tiếp; làm vậy sẽ khiến quy tắc nghiệp vụ bị trộn với HTTP.

### 4. Viết handler

Trong `internal/handler/`, thêm hàm nhận `http.ResponseWriter` và `*http.Request`. Hàm này đọc path, query hoặc body; gọi service; chuyển kết quả thành status code và JSON.

### 5. Đăng ký route

Trong `internal/server/router.go`, thêm route trỏ đến handler. Go 1.22 trở lên hỗ trợ route có biến path, ví dụ:

```go
mux.HandleFunc("GET /api/v1/users/{id}", users.GetByID)
```

Để lấy biến `id` từ request, handler dùng `r.PathValue("id")`.

### 6. Viết test

- Test service cho quy tắc nghiệp vụ: đầu vào hợp lệ và không hợp lệ.
- Test handler nếu cần kiểm tra status code, JSON hoặc cách đọc request.
- Chạy `go test ./...` hoặc `make test`.

## Checklist tự kiểm tra

- Handler chỉ xử lý HTTP, không tự chứa nhiều quy tắc nghiệp vụ.
- Service không phụ thuộc vào chi tiết như status code HTTP.
- Repository chịu trách nhiệm truy cập dữ liệu.
- Lỗi được xử lý rõ ràng; không bỏ qua lỗi bất ngờ.
- Có test cho hành vi quan trọng.
