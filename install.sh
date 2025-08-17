#!/usr/bin/env bash

# URL file thực thi
EXE_URL="https://github.com/nguyendangkin/chin-pak/releases/download/v1.0.1/chin"
# Thư mục cài đặt
INSTALL_DIR="$HOME/.local/bin"
# Đường dẫn file thực thi
EXE_PATH="$INSTALL_DIR/chin"

# Tạo thư mục nếu chưa tồn tại
mkdir -p "$INSTALL_DIR"

echo "Downloading chin..."
curl -L "$EXE_URL" -o "$EXE_PATH"
chmod +x "$EXE_PATH"

# Thêm vào PATH nếu chưa có
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    SHELL_RC="$HOME/.bashrc"
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_RC="$HOME/.zshrc"
    fi
    echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$SHELL_RC"
    echo "Added $INSTALL_DIR to PATH. Please restart your terminal or run 'source $SHELL_RC'."
fi

echo "Installation completed. You can now run 'chin'."
