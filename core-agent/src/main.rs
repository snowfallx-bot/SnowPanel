mod api;
mod config;
mod cron;
mod docker;
mod file;
mod process;
mod security;
mod service;
mod system;

use std::env;
use std::io;
use std::net::TcpListener;

fn main() -> io::Result<()> {
    let host = env::var("CORE_AGENT_HOST").unwrap_or_else(|_| "0.0.0.0".to_string());
    let port = env::var("CORE_AGENT_PORT").unwrap_or_else(|_| "50051".to_string());
    let addr = format!("{host}:{port}");

    let listener = TcpListener::bind(&addr)?;
    println!("core-agent bootstrap listening on {addr}");

    for incoming in listener.incoming() {
        match incoming {
            Ok(_) => {}
            Err(err) => eprintln!("core-agent incoming connection error: {err}"),
        }
    }

    Ok(())
}
