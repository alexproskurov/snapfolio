
# Snapfolio

Snapfolio is a photo-sharing application complete with user authentication, image uploads, and a database backend. This project was built as part of the "Web Development with Go" course by Jon Calhoun. Snapfolio offers a comprehensive platform for users to share and manage their photo collections.

## Table of Contents

- [Features](#features)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Usage](#usage)
- [Technologies Used](#technologies-used)

## Features

- **User Authentication**: Secure login and registration system.
- **Image Uploads**: Users can upload and manage their photos.
- **Database Integration**: Stores user data and photo information.

## Getting Started

### Prerequisites

- Go 1.16+
- Docker

### Installation

1. **Clone the Repository**:

    ```bash
    git clone https://github.com/alexproskurov/snapfolio.git
    cd snapfolio
    ```

2. **Build and Run with Docker**:

    ```bash
    docker-compose up --build
    ```

## Configuration

Configure environment variables by copying `.env.template` to `.env` and updating the values as needed.

## Usage

### Running the Server

Start the application with Docker:

```bash
docker-compose up
```

### Adding Content

Users can upload their photos directly through the application interface after registering and logging in.

## Technologies Used

- **Go**: Backend programming language
- **Tailwind CSS**: Utility-first CSS framework
- **Docker**: Containerization for easy deployment
- **PostgreSQL**: Database for storing user and photo data
