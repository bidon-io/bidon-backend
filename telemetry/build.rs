fn main() -> Result<(), Box<dyn std::error::Error>> {
    std::env::set_var("PROTOC", protoc_bin_vendored::protoc_bin_path()?);
    let out = std::path::PathBuf::from(std::env::var("OUT_DIR")?);
    prost_build::Config::new()
        .file_descriptor_set_path(out.join("events.binpb"))
        .compile_protos(
            &["../schemas/proto/org/bidon/telemetry/v1/events.proto"],
            &["../schemas/proto"],
        )?;
    println!("cargo:rerun-if-changed=../schemas/proto/org/bidon/telemetry/v1/events.proto");
    Ok(())
}
