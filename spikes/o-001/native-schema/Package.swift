// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "NativeSchemaSpike",
    products: [.library(name: "NativeSchemaSpike", targets: ["NativeSchemaSpike"])],
    targets: [
        .target(name: "NativeSchemaSpike"),
        .executableTarget(name: "NativeSchemaSpikeChecker", dependencies: ["NativeSchemaSpike"]),
        .testTarget(name: "NativeSchemaSpikeTests", dependencies: ["NativeSchemaSpike"])
    ]
)
