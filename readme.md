# Chin - Công cụ gói file và thư mục

Một công cụ gói file và thư mục đơn giản, nhanh chóng được viết bằng Go. Chin tạo ra các file gói với đuôi `.chin` và hỗ trợ chia nhỏ file lớn thành nhiều phần.

-   Gói/giải gói cơ bản
-   Hỗ trợ chia nhỏ file
-   Progress bar
-   Phát hiện file thiếu
-   Kiểm tra tính toàn vẹn dữ liệu

## Tính năng

-   ✅ **Gói file/thư mục đơn lẻ**
-   ✅ **Gói nhiều file/thư mục cùng lúc**
-   ✅ **Chia nhỏ file** khi dung lượng quá lớn
-   ✅ **Tự động giải gói** với kiểm tra tính toàn vẹn dữ liệu
-   ✅ **Phát hiện file bị thiếu** trong archive chia nhỏ
-   ✅ **Thanh tiến trình** hiển thị quá trình gói/giải gói
-   ✅ **Đa nền tảng** (Windows, Linux, macOS)

## Cài đặt\

### Windows

```bash
Set-ExecutionPolicy Bypass -Scope Process -Force; iex ((New-Object Net.WebClient).DownloadString('https://raw.githubusercontent.com/nguyendangkin/chin-pak/main/install.ps1'))
```

### Gói file/thư mục đơn lẻ

```bash
# Gói một file
chin document.txt
# Kết quả: document.chin

# Gói một thư mục
chin my-folder
# Kết quả: my-folder.chin
```

### Gói nhiều file/thư mục

```bash
# Gói nhiều file và thư mục
chin file1.txt file2.pdf folder1 folder2
# Kết quả: file1-all.chin
```

### Gói với chia nhỏ file

```bash
# Chia nhỏ thành các file tối đa 100MB
chin -mb 100 large-folder
# Kết quả: large-folder-1.chin, large-folder-2.chin, ...

# Chia nhỏ nhiều file
chin -mb 500 file1.txt folder1 folder2
# Kết quả: file1-all-1.chin, file1-all-2.chin, ...
```

### Giải gói

```bash
# Giải gói file thông thường
chin archive.chin

# Giải gói file chia nhỏ (từ bất kỳ part nào)
chin data-1.chin
chin data-5.chin  # Tự động tìm tất cả các part
```

## Xử lý lỗi

### Phát hiện file bị thiếu

```bash
# Nếu thiếu part file, sẽ báo lỗi:
chin data-1.chin
# Error: missing part files: data-[2,3].chin
```

### Kiểm tra tính toàn vẹn

-   Tự động kiểm tra từng part file có thể đọc được
-   Phát hiện dữ liệu bị corrupt
-   Cảnh báo nếu part cuối có kích thước bất thường

## Hiệu suất

-   **Gói nhanh**: Không sử dụng thuật toán gói phức tạp, ưu tiên tốc độ
-   **Ít RAM**: Xử lý từng file một, không load toàn bộ vào memory
-   **Progress bar**: Hiển thị tiến trình real-time
-   **Parallel processing**: Có thể mở rộng để xử lý song song

## Lưu ý quan trọng

2. **Đường dẫn tương đối**: Giữ nguyên cấu trúc thư mục gốc
3. **File split**: Các part file phải liên tục (1, 2, 3, ...)
4. **Platform**: Đường dẫn file được chuẩn hóa theo OS

## Troubleshooting

### Lỗi thường gặp

**"missing part files"**

```bash
# Đảm bảo tất cả part files có mặt
ls data-*.chin
# data-1.chin  data-2.chin  data-3.chin
```

**"corrupted data"**

```bash
# File có thể bị hỏng, thử gói lại
chin -mb 100 original-folder
```

**"failed to create directory"**

```bash
# Kiểm tra quyền ghi trong thư mục hiện tại
ls -la
```
