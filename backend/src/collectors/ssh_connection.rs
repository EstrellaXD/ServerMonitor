use std::sync::Arc;

use async_trait::async_trait;
use russh::keys::ssh_key;
use russh::{client, ChannelMsg};
use tokio::sync::Mutex;

use crate::error::CollectorError;

struct ClientHandler;

#[async_trait]
impl client::Handler for ClientHandler {
    type Error = russh::Error;

    async fn check_server_key(
        &mut self,
        _server_public_key: &ssh_key::PublicKey,
    ) -> Result<bool, Self::Error> {
        Ok(true)
    }
}

pub struct SshConnection {
    handle: Arc<Mutex<Option<client::Handle<ClientHandler>>>>,
    host: String,
    port: u16,
    username: String,
    password: Option<String>,
    key_path: Option<String>,
}

impl SshConnection {
    pub fn new(
        host: String,
        port: u16,
        username: String,
        password: Option<String>,
        key_path: Option<String>,
    ) -> Self {
        Self {
            handle: Arc::new(Mutex::new(None)),
            host,
            port,
            username,
            password,
            key_path,
        }
    }

    async fn ensure_connected(
        guard: &mut tokio::sync::MutexGuard<'_, Option<client::Handle<ClientHandler>>>,
        host: &str,
        port: u16,
        username: &str,
        password: &Option<String>,
        key_path: &Option<String>,
    ) -> Result<(), CollectorError> {
        // Check existing connection
        if let Some(ref handle) = **guard {
            if !handle.is_closed() {
                return Ok(());
            }
        }
        **guard = None;

        let config = Arc::new(client::Config {
            inactivity_timeout: Some(std::time::Duration::from_secs(30)),
            ..Default::default()
        });

        let handler = ClientHandler;
        let mut handle = client::connect(config, (host, port), handler)
            .await
            .map_err(|e| {
                CollectorError::Ssh(format!("Connect to {host}:{port} failed: {e}"))
            })?;

        let authenticated = if let Some(ref kp) = key_path {
            let key_pair = russh_keys::load_secret_key(kp, None)
                .map_err(|e| CollectorError::Ssh(format!("Failed to load key: {e}")))?;
            handle
                .authenticate_publickey(username, Arc::new(key_pair))
                .await
                .map_err(|e| CollectorError::Ssh(format!("Auth failed: {e}")))?
        } else if let Some(ref pw) = password {
            handle
                .authenticate_password(username, pw)
                .await
                .map_err(|e| CollectorError::Ssh(format!("Auth failed: {e}")))?
        } else {
            return Err(CollectorError::Ssh(
                "No authentication method provided".to_string(),
            ));
        };

        if !authenticated {
            return Err(CollectorError::Ssh("Authentication failed".to_string()));
        }

        **guard = Some(handle);
        Ok(())
    }

    pub async fn run_command(&self, cmd: &str) -> Result<String, CollectorError> {
        let mut guard = self.handle.lock().await;

        Self::ensure_connected(
            &mut guard,
            &self.host,
            self.port,
            &self.username,
            &self.password,
            &self.key_path,
        )
        .await?;

        let handle = guard.as_ref().unwrap();

        let mut channel = handle
            .channel_open_session()
            .await
            .map_err(|e| CollectorError::Ssh(format!("Channel open failed: {e}")))?;

        channel
            .exec(true, cmd)
            .await
            .map_err(|e| CollectorError::Ssh(format!("Exec failed: {e}")))?;

        let mut output = Vec::new();
        while let Some(msg) = channel.wait().await {
            match msg {
                ChannelMsg::Data { data } => {
                    output.extend_from_slice(&data);
                }
                ChannelMsg::Eof | ChannelMsg::Close => break,
                _ => {}
            }
        }

        String::from_utf8(output)
            .map_err(|e| CollectorError::Ssh(format!("UTF-8 decode error: {e}")))
    }

    pub async fn close(&self) {
        let mut guard = self.handle.lock().await;
        if let Some(handle) = guard.take() {
            let _ = handle
                .disconnect(russh::Disconnect::ByApplication, "", "en")
                .await;
        }
    }
}
