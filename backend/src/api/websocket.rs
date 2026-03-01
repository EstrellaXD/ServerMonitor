use std::sync::Arc;

use axum::extract::ws::{Message, WebSocket};
use axum::extract::{State, WebSocketUpgrade};
use axum::response::IntoResponse;
use futures_util::{SinkExt, StreamExt};

use crate::api::routes::AppState;

pub async fn ws_handler(
    ws: WebSocketUpgrade,
    State(state): State<Arc<AppState>>,
) -> impl IntoResponse {
    let update = state.store.create_update_message().await;
    let initial = serde_json::to_string(&update).unwrap_or_default();
    ws.on_upgrade(move |socket| handle_socket(socket, initial, state))
}

async fn handle_socket(socket: WebSocket, initial_message: String, state: Arc<AppState>) {
    let (mut sender, mut receiver) = socket.split();

    // Send initial state
    if sender
        .send(Message::Text(initial_message.into()))
        .await
        .is_err()
    {
        return;
    }

    // Subscribe to broadcast
    let mut rx = state.broadcast_tx.subscribe();

    // Forward broadcasts to this client
    let mut send_task = tokio::spawn(async move {
        while let Ok(msg) = rx.recv().await {
            if sender.send(Message::Text(msg.into())).await.is_err() {
                break;
            }
        }
    });

    // Handle incoming messages (keepalive)
    let mut recv_task = tokio::spawn(async move {
        while let Some(Ok(msg)) = receiver.next().await {
            match msg {
                Message::Text(text) if text.as_str() == "ping" => {}
                Message::Close(_) => break,
                _ => {}
            }
        }
    });

    tokio::select! {
        _ = &mut send_task => recv_task.abort(),
        _ = &mut recv_task => send_task.abort(),
    }
}
