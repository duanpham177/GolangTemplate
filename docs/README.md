# Tài liệu ServerGO

Tài liệu này giải thích project theo thứ tự phù hợp cho người mới học Go. Bạn không cần đọc hết một lần; hãy bắt đầu từ cấu trúc, sau đó lần theo một request mẫu.

## Lộ trình đọc

1. [Tổng quan cấu trúc project](project-structure.md) — mỗi thư mục và file dùng làm gì.
2. [Luồng xử lý request](request-lifecycle.md) — một request đi từ router đến dữ liệu rồi quay lại như thế nào.
3. [Thêm API mới](adding-feature.md) — các bước thực hành để tự mở rộng project.
4. [Chạy và kiểm thử](running-testing.md) — khởi động server, gọi API và chạy test.
5. [Làm việc offline với vendor](offline-vendor.md) — chuẩn bị dependency khi máy dev bị cách ly mạng.
6. [TCP thuần cho IoT](tcp-iot.md) — giao thức, kết nối thiết bị và API đọc/gửi lệnh.
7. [Thuật ngữ cơ bản](glossary.md) — giải thích ngắn các từ thường gặp trong code.

## Nguyên tắc của template

- `handler` giao tiếp với HTTP.
- `service` chứa quy tắc nghiệp vụ.
- `repository` chịu trách nhiệm lưu và lấy dữ liệu.
- `model` mô tả dữ liệu.
- `transport/tcp` nhận dữ liệu và gửi lệnh qua TCP.
- `cmd/server/main.go` khởi tạo và nối các thành phần.

Không phải ứng dụng nào cũng cần đủ mọi thư mục. Hãy thêm cấu trúc khi project thực sự cần, thay vì tạo nhiều lớp chỉ để làm theo khuôn.
