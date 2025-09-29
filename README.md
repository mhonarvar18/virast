# Virast - Social Timeline Project

This project is a scalable social network with a **Timeline** feature, built with **Go**, **GORM**, **Redis**, and **MySQL**.  
The goal is to efficiently manage millions of users and posts using a **Fan-out Worker**.

---

## 🔹 Features

- **Post Service**: Create new posts by users and add them to the Fanout queue.  
- **Timeline Service**: Fetch user timelines from Redis with Pagination support (`start` / `limit`).  
- **Fan-out Worker**: Distribute posts to followers’ timelines in batches and store records in the `timeline` table.  
- **Repository Pattern / Hexagonal Architecture** for testability and scalability.  
- **Redis ZSET** for fast timeline retrieval ordered by post timestamp.  
- **MySQL** using GORM for data storage.  

---

## 🔹 Prerequisites

- Go >= 1.21  
- MySQL / MariaDB  
- Redis  
- Git  

---

## 🔹 Installation and Setup

### 1. Clone the repository
```bash
git clone https://github.com/yourusername/virast.git
cd virast
```

install dependency:
```bash
go mod tidy
```

Install GORM v2
```bash
go get -u gorm.io/gorm
go get -u gorm.io/driver/mysql

```

Run the project
```bash
go run cmd/app/main.go
```

Stability / Stress Test

You can generate many fake users and posts for testing (just uncomment testStability method in main.go):
```m
testStability(ctx, userSvc, postSvc, followerScv)
```

