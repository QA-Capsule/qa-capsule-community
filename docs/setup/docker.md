---
icon: fontawesome/brands/docker
---

# Docker Deployment

QA Capsule is designed to be deployed effortlessly using Docker and Docker Compose. This ensures that the application runs identically on your local machine, a staging server, or a production environment.

## Prerequisites

Ensure you have the following installed on your host machine:

* **Docker** (v20.10 or newer)
* **Docker Compose** (v2.0 or newer)

## Installation Steps

### 1. Clone the Repository :
   
```bash
git clone https://github.com/Ashraf-Khabar/qa-capsule
cd qa-capsule
```

### 2. Understand the Docker Compose File :

* The `docker-compose.yml` file is configured to build the Go backend and expose the web interface.

* Crucial Note on Persistence : `QA Capsule` uses `SQLite` for its database (`qacapsule.db`). The `docker-compose.yml` maps a volume to the `./data` directory. Never delete this folder in production, or you will lose all your users, organizations, and project configurations.

### 3. Start the Application : 

Run the following command to build the Go binary and start the container in detached mode :

```bash
docker-compose up -d --build
```
### 4. Verify the Deployment : 

Open your web browser and navigate to: http://localhost:9000. You should see the QA Capsule login screen.

## Stopping and Updating

To gracefully stop the application without losing data:

```bash
docker-compose down
```

To update to the latest version, pull the repository changes and rebuild:

```Bash
git pull origin main
docker-compose up -d --build
```