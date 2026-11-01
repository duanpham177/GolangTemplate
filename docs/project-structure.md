# Tổng quan cấu trúc project

Project này có HTTP API mẫu và TCP server để thiết bị IoT gửi telemetry/nhận lệnh. Đây là một cách tổ chức phổ biến, không phải quy tắc bắt buộc cho mọi dự án Go.

## Cây thư mục

```text
ServerGO/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── handler/
│   ├── model/
│   ├── repository/
│   ├── server/
│   ├── service/
│   └── transport/
│       └── tcp/
├── docs/
├── configs/
├── migrations/
├── pkg/
├── .env.example
├── .gitignore
├── go.mod
├── Makefile
└── README.md
```

## Những nơi quan trọng

### `cmd/server/`

Chứa chương trình có thể chạy. [main.go](../cmd/server/main.go) là điểm bắt đầu của ứng dụng: đọc cấu hình, tạo logger, nối các thành phần, mở HTTP server và tắt server khi nhận tín hiệu dừng.

Nếu sau này ứng dụng có thêm chương trình nền, chẳng hạn worker xử lý hàng đợi, có thể tạo thêm `cmd/worker/`. Mỗi thư mục trong `cmd/` thường tương ứng với một chương trình riêng.

### `internal/`

Chứa code nội bộ của module. Go ngăn các module bên ngoài import package bên trong `internal/`. Trong project này, các package bên trong vẫn có thể gọi nhau.

- `config/`: đọc cổng, token API, token thiết bị và cấu hình TLS từ biến môi trường.
- `handler/`: nhận request HTTP, gọi service và trả response.
- `model/`: định nghĩa các kiểu dữ liệu, ví dụ `User`.
- `repository/`: định nghĩa cách truy cập dữ liệu; hiện tại lưu trong bộ nhớ.
- `server/`: đăng ký HTTP route và bọc router bằng middleware log.
- `service/`: xử lý quy tắc nghiệp vụ dùng chung cho HTTP và TCP.
- `transport/tcp/`: nhận kết nối TCP, xác thực thiết bị, đọc JSON từng dòng và gửi lệnh đến thiết bị đang online.

### Các file ở thư mục gốc

- `go.mod`: tên module và phiên bản Go tối thiểu. Tên module `servergo` cũng được dùng trong các câu lệnh import như `servergo/internal/service`.
- `Makefile`: các lệnh tắt như `make run`, `make test` và `make vendor`.
- `.env.example`: ví dụ về các biến môi trường. Đây chỉ là file mẫu; code hiện tại không tự đọc file `.env`.
- `.gitignore`: liệt kê file Git không nên đưa vào repository.
- `README.md`: trang giới thiệu ngắn và cách bắt đầu.
- `docs/`: tài liệu chi tiết hơn để học và tra cứu.

## Các thư mục tùy chọn

- `pkg/`: dành cho package mà bạn chủ đích muốn cho module khác sử dụng. Không cần thêm code vào đây nếu project chưa có nhu cầu đó.
- `configs/`: có thể dùng để chứa file cấu hình không bí mật nếu ứng dụng cần.
- `migrations/`: thường chứa các file SQL tạo hoặc thay đổi bảng database. Template hiện chưa kết nối database nên thư mục này đang trống.
- `vendor/`: chứa source của dependency bên ngoài để build offline. Template hiện chỉ dùng thư viện chuẩn nên chưa cần thư mục này. Xem [hướng dẫn vendor](offline-vendor.md).

## Vì sao tách thành nhiều tầng?

Mỗi tầng có một trách nhiệm:

```text
HTTP handler ─┐
              ├──> service -> repository
TCP transport ┘
```

Nhờ vậy, quy tắc nghiệp vụ và cách lưu dữ liệu không phụ thuộc vào giao thức. HTTP và TCP có thể cùng gọi service; khi thay nơi lưu dữ liệu, mục tiêu là thay repository mà không phải viết lại các giao thức.

Không nên tạo một package hoặc tầng riêng nếu nó chưa có trách nhiệm rõ ràng. Với ứng dụng nhỏ, code đơn giản thường dễ học và bảo trì hơn.
