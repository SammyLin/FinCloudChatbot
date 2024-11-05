# Multi-Callback LINE Bot Server

A Go server that supports multiple LINE Bot callbacks, enabling simultaneous handling of messages from different LINE Bots with various callback types.

## Features

- Multiple LINE Bot callback configurations
- Dynamic route registration
- Support for different callback types:
  - bypass: Forward messages to external API with customizable endpoints
  - periodicsummary: Handle periodic summary functionality
- Comprehensive logging system
- Development mode with debug logging
- Ngrok integration for local webhook testing

## Tech Stack

- Go 1.16+
- LINE Messaging API SDK v8
- Ngrok for local development
- Environment-based configuration

## Prerequisites

- Go 1.16 or higher
- Ngrok installed locally
- LINE Developer Account
- LINE Channel Secret and Access Token
