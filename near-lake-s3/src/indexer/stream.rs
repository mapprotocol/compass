use std::collections::HashMap;
use std::process::id;
use crate::config::{init_lake_config, PROJECT_CONFIG, update_synced_block_height, redis_publisher, INDEXER, REDIS};
use futures::StreamExt;
use serde_json::json;
use near_lake_framework::near_indexer_primitives::views::{ExecutionOutcomeWithIdView, ExecutionStatusView};

pub async fn indexer_stream_from_s3() {
    let config = init_lake_config().await;

    let (_, stream) = near_lake_framework::streamer(config);

    let mut handlers = tokio_stream::wrappers::ReceiverStream::new(stream)
        .map(handle_streamer_message)
        .buffer_unordered(1usize);

    while let Some(_handle_message) = handlers.next().await {}
}

pub async fn handle_streamer_message(
    streamer_message: near_lake_framework::near_indexer_primitives::StreamerMessage,
) {
    tracing::info!("Block height {}", streamer_message.block.header.height);

    let mut publish = false;
    let mut receipt_id2tx_id: HashMap<String, String> = HashMap::new();

    'outer: for shard in &streamer_message.shards {
        for tx_res in &shard.receipt_execution_outcomes {
            if is_valid_receipt(&tx_res.execution_outcome) {
                publish = true;

                let tx_id_opt = redis_publisher().get(tx_res.execution_outcome.id.to_string().as_str()).await;
                if let Some(tx_id) = tx_id_opt {
                    for receipt_id in &tx_res.execution_outcome.outcome.receipt_ids {
                        receipt_id2tx_id.insert(receipt_id.to_string(), tx_id.clone());
                    }
                }
            }
        }
    }

    for shard in &streamer_message.shards {
        if let Some(chunk) = &shard.chunk {
            for tx in &chunk.transactions {
                if PROJECT_CONFIG.accounts.contains(&tx.transaction.receiver_id.to_string()) {
                    if let ExecutionStatusView::SuccessReceiptId(successReceiptId) = tx.outcome.execution_outcome.outcome.status {
                        receipt_id2tx_id.insert(successReceiptId.to_string(), tx.transaction.hash.to_string());
                    }
                }
            }
        }
    }

    for (receipt_id, tx_id) in receipt_id2tx_id {
        redis_publisher().set(receipt_id.clone().as_str(), tx_id.clone()).await;
        tracing::info!(
            target: INDEXER,
            "Save Receipt ID {} / TX ID {} on block {}",
            receipt_id,
            tx_id,
            streamer_message.block.header.height
            );
    }

    if publish {
        let json = json!(streamer_message).to_string();
        redis_publisher().lpush(json).await;
        update_synced_block_height(streamer_message.block.header.height).await;

        tracing::info!(
            target: INDEXER,
            "Save {} / shards {}",
            streamer_message.block.header.height,
            streamer_message.shards.len()
        );
    } else {
        if streamer_message.block.header.height % 100 == 0 {
            update_synced_block_height(streamer_message.block.header.height).await;
            tracing::info!(
                target: REDIS,
                "Update synced block height {}",
                streamer_message.block.header.height
            )
        }
    }
}

pub fn is_valid_receipt(execution_outcome: &ExecutionOutcomeWithIdView) -> bool {
    match &execution_outcome.outcome.status {
        ExecutionStatusView::Unknown => return false,
        _ => ()
    }

    PROJECT_CONFIG.accounts.contains(&execution_outcome.outcome.executor_id.to_string())
}
