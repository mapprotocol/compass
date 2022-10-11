use std::path::PathBuf;
use std::str::FromStr;
use anyhow::Result;
use dotenv::dotenv;
use tracing::Level;
use tracing_appender::rolling::RollingFileAppender;
use crate::config::{init_redis_pusher, PROJECT_CONFIG};
use crate::indexer::stream::indexer_stream_from_s3;

pub mod indexer;
pub mod pusher;
pub mod config;


#[tokio::main]
async fn main() -> Result<(), tokio::io::Error> {
    dotenv().ok();

    let path = PathBuf::from_str(&PROJECT_CONFIG.log_file).unwrap();
    let file_appender : RollingFileAppender  = tracing_appender::rolling::daily(path.parent().unwrap(), path.file_name().unwrap());
    let (non_blocking, _guard) = tracing_appender::non_blocking(file_appender);
    tracing_subscriber::fmt()
        .with_max_level(Level::from_str(&PROJECT_CONFIG.log_level).unwrap())
        .with_writer(non_blocking.clone())
        .init();
    tracing::info!(".tracing is initialized");

    init_redis_pusher().await;
    tracing::info!(".redis pusher is initialized");

    indexer_stream_from_s3().await;

    Ok(())
}