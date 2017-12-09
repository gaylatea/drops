load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "server",
    srcs = ["handler.go", "server.go"],
    importpath = "github.com/silversupreme/drops",
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_golang_glog//:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ]
)