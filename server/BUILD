load("@io_bazel_rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "server",
    srcs = ["main.go", "cmds.go"],
    importpath = "github.com/silversupreme/drops/server",
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_golang_glog//:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ]
)