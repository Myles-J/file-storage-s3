# File Storage S3 Backend

A Go-based backend for file and video storage with user authentication, video uploads, thumbnail management, and S3-compatible storage support. This project is suitable for learning, prototyping, or as a foundation for a media storage service.

## Features

- User registration and authentication (JWT-based)
- Video metadata management (create, retrieve, delete)
- Video and thumbnail upload endpoints
- SQLite database for metadata
- S3-compatible storage integration (configurable)
- RESTful API endpoints
- Simple web frontend (in `app/`)

## Requirements

- Go 1.23+
- SQLite3
- (Optional) AWS S3 or compatible storage

## Setup & Installation

1. **Clone the repository:**
   ```sh
   git clone <repo-url>
   cd file-storage-s3
   ```
2. **Install dependencies:**
   ```sh
   go mod download
   ```
3. **Configure environment variables:**
   Create a `.env` file in the project root with the following variables:
   ```env
   DB_PATH=./tubely.db
   JWT_SECRET=your_jwt_secret
   PLATFORM=local
   FILEPATH_ROOT=./app
   ASSETS_ROOT=./assets
   S3_BUCKET=your-s3-bucket
   S3_REGION=your-s3-region
   S3_CF_DISTRO=your-cloudfront-distribution
   PORT=8080
   ```
   Adjust values as needed for your environment.

4. **Run the server:**
   ```sh
   go run main.go
   ```
   The server will start on `http://localhost:8080/app/` by default.

## Usage

- Access the web frontend at `/app/` for uploading and managing videos.
- API endpoints are available under `/api/`.

## API Endpoints

| Method | Endpoint                        | Description            |
| ------ | ------------------------------- | ---------------------- |
| POST   | /api/login                      | User login             |
| POST   | /api/refresh                    | Refresh JWT            |
| POST   | /api/revoke                     | Revoke refresh token   |
| POST   | /api/users                      | Register new user      |
| POST   | /api/videos                     | Create video metadata  |
| GET    | /api/videos                     | List user's videos     |
| GET    | /api/videos/{videoID}           | Get video metadata     |
| DELETE | /api/videos/{videoID}           | Delete video           |
| POST   | /api/thumbnail_upload/{videoID} | Upload thumbnail       |
| POST   | /api/video_upload/{videoID}     | Upload video file      |
| GET    | /api/thumbnails/{videoID}       | Get video thumbnail    |
| POST   | /admin/reset                    | Reset database (admin) |

## Sample Data

Run `./samplesdownload.sh` to download sample images and videos into the `samples/` directory.

## License

[MIT](LICENSE) (or specify your license here)