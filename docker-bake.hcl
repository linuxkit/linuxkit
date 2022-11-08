group "default" {
  targets = ["binaries"]
}

target "binaries" {
  target = "binaries"
  output = ["./bin/build"]
  platforms = [
    "linux/amd64",
    "linux/arm64",
    "windows/amd64",
    "darwin/amd64",
    "darwin/arm64",
  ]
}
