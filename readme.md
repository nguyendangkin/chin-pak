# 📦 CHIN PACKER

Một công cụ đóng gói và giải gói dữ liệu được viết bằng Go, hỗ trợ đóng gói nhiều file/thư mục và chia nhỏ file đóng gói thành các phần.

## ✨ Tính năng

-   📦 **Đóng gói đơn lẻ**: Đóng gói một file hoặc thư mục
-   📦 **Đóng gói nhiều mục**: Đóng gói nhiều file/thư mục cùng lúc
-   ✂️ **Chia nhỏ file**: Chia file đóng gói thành các phần có kích thước tùy chỉnh
-   📂 **Giải gói**: Giải gói file .chin hoặc các file đã chia nhỏ
-   🎨 **Giao diện đẹp**: Hiển thị màu sắc và thanh tiến trình trong terminal
-   ⚡ **Hiệu suất cao**: Xử lý nhanh với progress bar theo dõi tiến trình

## 🚀 Cài đặt

-   Cho Windows

```bash
Set-ExecutionPolicy Bypass -Scope Process -Force; iex ((New-Object Net.WebClient).DownloadString('https://raw.githubusercontent.com/nguyendangkin/chin-pak/main/install.ps1'))
```

## 📖 Hướng dẫn sử dụng

### Đóng gói dữ liệu

#### Đóng gói một file hoặc thư mục:

```bash
chin <file/folder>
```

**Ví dụ:**

```bash
chin document.txt          # Tạo ra document.chin
chin my-folder/            # Tạo ra my-folder.chin
```

#### Đóng gói nhiều file/thư mục:

```bash
chin <file1> <file2> <folder1> ...
```

**Ví dụ:**

```bash
chin file1.txt file2.txt folder1/ folder2/
# Tạo ra file1-all.chin
```

#### Đóng gói với tính năng chia nhỏ:

```bash
chin -mb <kích_thước_MB> <file/folder>
```

**Ví dụ:**

```bash
chin -mb 100 large-folder/
# Tạo ra: large-folder-1.chin, large-folder-2.chin, ...
```

### Giải gói dữ liệu

#### Giải gói file thông thường:

```bash
chin file.chin
```

#### Giải gói file đã chia nhỏ:

```bash
chin file-1.chin
# Tự động tìm và kết hợp tất cả các phần: file-1.chin, file-2.chin, ...
```

## 💡 Ví dụ thực tế

### Đóng gói một dự án web:

```bash
chin my-website/
# Output: my-website.chin
```

### Đóng gói nhiều file cấu hình:

```bash
chin config.json settings.ini database.sql
# Output: config-all.chin
```

### Đóng gói và chia nhỏ file lớn:

```bash
chin -mb 50 backup-data/
# Output: backup-data-1.chin, backup-data-2.chin, backup-data-3.chin, ...
```

### Giải gói:

```bash
chin my-website.chin        # Giải gói file thông thường
chin backup-data-1.chin     # Giải gói file đã chia nhỏ
```

## 📋 Cấu trúc file .chin

File .chin sử dụng format binary tùy chỉnh:

-   Header chứa độ dài đường dẫn (uint16)
-   Đường dẫn file/thư mục
-   Độ dài dữ liệu (uint32)
-   Dữ liệu file (nếu là file) hoặc 0 bytes (nếu là thư mục)

## 🎯 Tùy chọn dòng lệnh

| Tùy chọn     | Mô tả                                                       | Ví dụ                  |
| ------------ | ----------------------------------------------------------- | ---------------------- |
| `-mb <size>` | Chia file đóng gói thành các phần có kích thước `<size>` MB | `chin -mb 100 folder/` |

## 🔧 Yêu cầu hệ thống

-   Go 1.16 trở lên
-   Hệ điều hành: Windows, macOS, Linux
-   Terminal hỗ trợ ANSI colors (cho hiển thị màu sắc)

## 📊 Thông tin hiển thị

Chương trình hiển thị các thông tin chi tiết:

-   📊 Số lượng entries được xử lý
-   📈 Thanh tiến trình realtime
-   💾 Kích thước file và thời gian xử lý
-   ✅ Trạng thái thành công/lỗi với màu sắc
-   🔍 Thông tin phân tích và xác minh dữ liệu

## ⚠️ Lưu ý

1. **File output**: Luôn được tạo trong thư mục hiện tại
2. **Đường dẫn tương đối**: Được bảo toàn trong file đóng gói
3. **File chia nhỏ**: Phải có đầy đủ tất cả các phần để giải gói
4. **Quyền truy cập**: Đảm bảo có quyền đọc/ghi trong thư mục làm việc

## 🐛 Xử lý lỗi

-   Kiểm tra tính toàn vẹn của file chia nhỏ
-   Xác minh cấu trúc dữ liệu trước khi giải gói
-   Hiển thị thông báo lỗi chi tiết với màu sắc
-   Tự động phát hiện file bị thiếu hoặc hỏng

## 📞 Hỗ trợ

Nếu gặp vấn đề, hãy kiểm tra:

1. Quyền truy cập file/thư mục
2. Dung lượng ổ đĩa còn trống
3. Tính toàn vẹn của file .chin
4. Đầy đủ các file phần (đối với file chia nhỏ)

---

_CHIN Packer - Công cụ đóng gói dữ liệu mạnh mẽ và thân thiện với người dùng! 🚀_
