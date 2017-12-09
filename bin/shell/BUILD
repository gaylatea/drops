load("@io_bazel_rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "shell",
    srcs = ["main.go"],
    importpath = "github.com/silversupreme/drops/shell",
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_golang_glog//:go_default_library",
    ]
)