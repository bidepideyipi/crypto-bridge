Crypto Bridge - Multi-Chain Wallet System

  A production-grade cryptocurrency wallet service that bridges blockchain networks with centralized exchange systems. Built with Go, supporting Bitcoin and Ethereum with a focus on security, reliability, and scalability.

  Features

  - Multi-Chain Support - Bitcoin (BTC) and Ethereum (ETH) adapters with extensible architecture
  - Deposit Monitoring - Real-time blockchain event listening with configurable confirmation thresholds
  - Withdrawal Management - Automated and manual approval workflows with cold/hot wallet separation
  - Security First - HSM integration, Shamir Secret Sharing, and K-of-N multi-sig support
  - Reconciliation - Automatic balance verification and audit trails
  - High Concurrency - Goroutine-per-Chain model for independent chain processing

  Tech Stack

  - Language: Go 1.18+
  - Blockchain: btcd (BTC), go-ethereum (ETH)
  - Database: PostgreSQL (primary) + Redis (cache)
  - Message Queue: RocketMQ
  - Architecture: Layered service design with clear separation of concerns

  Documentation

  - 架构设计
  - 接口定义
  - 数据库设计
  - 测试计划

  Development Roadmap

  - [x] BTC chain adapter
  - [x] Deposit monitoring service
  - [x] Address management
  - [ ] ETH chain adapter
  - [ ] Cold wallet service integration
  - [ ] TRX/SOL support

  License

  MIT
