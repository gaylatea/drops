load("@io_bazel_rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "server",
    srcs = ["main.go"],
    importpath = "github.com/silversupreme/drops/bin/server",
    visibility = ["//visibility:public"],
    deps = [
        "//:server",
        "@com_github_golang_glog//:go_default_library",
    ]
)