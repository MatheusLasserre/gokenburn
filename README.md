# Gokenburn

A simple script to scale an image up and generate a video from it using ffmpeg.

Fastest frame generation time: ~9ms

## Setup and Installation

1. Clone the repository:

    ```
    git clone https://github.com/MatheusLasserre/gokenburn.git
    ```

2. Download dependencies:

    ```
    go mod tidy
    ```

3. Install ffmpeg on your system.

## Running the Project

1. By default, it is configured to the jpg image in the assets folder: "example.jpg", and to out the video at the same folder as "example-scaled.mp4".

2. Run the main script:
   ```
   go run main.go
   ```