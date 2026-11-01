# Thuật ngữ cơ bản

| Thuật ngữ | Giải thích ngắn |
|---|---|
| API | Cách một chương trình cung cấp chức năng để chương trình khác gọi. |
| HTTP request | Yêu cầu gửi đến server, thường gồm phương thức, đường dẫn, header và body. |
| HTTP response | Câu trả lời từ server, gồm status code, header và có thể có body. |
| Route | Quy tắc nối phương thức và đường dẫn HTTP với một hàm xử lý. |
| Handler | Hàm nhận request HTTP và tạo response HTTP. |
| Service | Tầng chứa quy tắc nghiệp vụ của ứng dụng. |
| Repository | Tầng đọc và ghi dữ liệu; dữ liệu có thể ở bộ nhớ hoặc database. |
| Model | Kiểu dữ liệu mô tả đối tượng của ứng dụng, ví dụ `User`. |
| Middleware | Hàm bọc handler/router để thực hiện công việc chung trước hoặc sau request. |
| Module | Đơn vị quản lý code và dependency của Go; project này có module tên `servergo`. |
| Package | Nhóm file Go trong một thư mục, thường có cùng khai báo `package`. |
| Dependency | Module bên ngoài mà project cần dùng. |
| Vendor | Cách lưu source dependency trong project để có thể build mà không tải từ mạng. |
| JSON | Định dạng văn bản thường dùng để trao đổi dữ liệu giữa client và API. |
| Token | Chuỗi bí mật dùng như mật khẩu để chứng minh người gọi được phép truy cập. Xem [Token và xác thực](running-testing.md#token-và-xác-thực). |
| Bearer token | Cách gửi token trong header HTTP: `Authorization: Bearer <token>`. |
| Unit test | Test một phần nhỏ của chương trình, như một hàm hoặc service. |
| Graceful shutdown | Dừng server có trật tự, cho request đang chạy cơ hội hoàn tất. |
