git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    tag = "0.6.0",
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains", "go_repository")
go_rules_dependencies()
go_register_toolchains()

go_repository(
    name = "com_github_golang_glog",
    tag = "master",
    importpath = "github.com/golang/glog",
)

go_repository(
    name = "com_github_pkg_errors",
    tag = "master",
    importpath = "github.com/pkg/errors",
)
