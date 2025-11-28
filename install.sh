go build -o dewormer

# On macos
if [[ "$OSTYPE" == "darwin"* ]]; then
  sudo mv dewormer /usr/local/bin/
  exit 0
fi

# On linux
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
  sudo mv dewormer /usr/local/bin/
  exit 0
fi

# On windows (using git bash)
if [[ "$OSTYPE" == "msys"* || "$OSTYPE" == "cygwin"* ]]; then
  mv dewormer.exe /c/Windows/System32/
  exit 0
fi

# Test that the installation was successful
dewormer --version